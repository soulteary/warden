# Documentazione d'uso dell'SDK

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](../zhCN/SDK.md) | [Français](../frFR/SDK.md) | [Italiano](SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](../deDE/SDK.md) | [한국어](../koKR/SDK.md)

Warden fornisce un SDK in Go per facilitare l'integrazione in altri progetti. L'SDK offre un'interfaccia API essenziale con supporto per cache, autenticazione e altro ancora.

## Caratteristiche

- 🚀 **Semplice e immediato**: offre interfacce API essenziali
- ⚡ **Prestazioni elevate**: cache integrata (GetUsers); le query dirette (GetUserByIdentifier) riducono le chiamate all'API
- 🔒 **Sicuro**: supporta l'autenticazione tramite chiave API; la gestione degli errori non espone informazioni sensibili
- 📦 **Flessibile**: timeout, TTL della cache e altro sono configurabili
- 🔌 **Estendibile**: supporta implementazioni di logger personalizzate
- 🎯 **Fallback intelligente**: CheckUserInList ripiega automaticamente sull'e-mail quando il telefono non viene trovato

## Installare l'SDK

```bash
go get github.com/soulteary/warden/pkg/warden
```

## Avvio rapido

### Uso di base

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Creare le opzioni del client
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)
    
    // Creare il client
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    // Recuperare l'elenco degli utenti
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }
    
    // Verificare se l'utente è nell'elenco (phone, mail o entrambi)
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // È possibile usare anche solo phone oppure solo mail
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // Recuperare i dettagli dell'utente
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### Usare un logger personalizzato

L'SDK supporta implementazioni di logger personalizzate, per esempio con logrus:

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    logger := logrus.StandardLogger()
    
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithLogger(warden.NewLogrusAdapter(logger))
    
    client, err := warden.NewClient(opts)
    // ...
}
```

### Query paginata

```go
// Recuperare l'elenco paginato degli utenti
resp, err := client.GetUsersPaginated(ctx, 1, 10) // Pagina 1, 10 elementi per pagina
if err != nil {
    panic(err)
}

fmt.Printf("Total users: %d\n", resp.Pagination.Total)
fmt.Printf("Total pages: %d\n", resp.Pagination.TotalPages)
for _, user := range resp.Data {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
}
```

### Recuperare le informazioni di un singolo utente

```go
// Recuperare le informazioni dell'utente tramite telefono
user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
        println("User not found")
    } else {
        panic(err)
    }
} else {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
    if user.IsActive() {
        println("User is active")
    }
}

// Recuperare le informazioni dell'utente tramite e-mail
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// Recuperare le informazioni dell'utente tramite ID utente
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
```

### Svuotare la cache

```go
// Svuotare manualmente la cache del client
client.ClearCache()

// Oppure usare l'alias
client.InvalidateCache()
```

### Trasporto HTTP personalizzato

```go
import "net/http"

// Creare un trasporto personalizzato
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### Firma delle richieste con HMAC v2

```go
opts := warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET"))

client, err := warden.NewClient(opts)
```

L'SDK firma l'ID della chiave, il timestamp, il nonce, il percorso/query, il metodo e l'hash del corpo.
Sia l'ID della chiave sia il segreto devono essere configurati; una coppia incompleta restituisce
`ErrCodeInvalidConfig` anziché inviare una richiesta non firmata.

### Configurazione dei tentativi ripetuti

```go
// Configurare le opzioni dei tentativi ripetuti
retryOpts := warden.DefaultRetryOptions()
retryOpts.MaxRetries = 3
retryOpts.RetryDelay = 100 * time.Millisecond
retryOpts.MaxRetryDelay = 5 * time.Second
retryOpts.BackoffMultiplier = 2.0

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithRetry(retryOpts)

client, err := warden.NewClient(opts)
```

### Invalidazione della cache guidata dagli eventi

```go
// Creare un canale per gli eventi di invalidazione della cache
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // Importante: chiudere per fermare il listener in background

// In seguito, attivare l'invalidazione della cache da un evento esterno
invalidationCh <- struct{}{}

// La cache verrà svuotata automaticamente alla ricezione del segnale
```

## Riferimento dell'API

### Options

La struttura `Options` serve a configurare il client:

- `BaseURL`: indirizzo del servizio Warden (obbligatorio)
- `APIKey`: chiave API (facoltativa)
- `Timeout`: timeout delle richieste HTTP (predefinito 10 secondi)
- `CacheTTL`: TTL della cache (predefinito 5 minuti)
- `Logger`: interfaccia del logger (facoltativa, per impostazione predefinita NoOpLogger)
- `Transport`: trasporto HTTP personalizzato (facoltativo)
- `HMACKeyID` / `HMACSecret`: coppia di firma HMAC v2 (entrambi o nessuno)
- `TLSConfig`: configurazione client TLS/mTLS (facoltativa)
- `Retry`: configurazione dei tentativi ripetuti (facoltativa, per impostazione predefinita nessun tentativo)
- `CacheInvalidationChannel`: canale per l'invalidazione della cache guidata dagli eventi (facoltativo)

### Metodi del client

#### `NewClient(opts *Options) (*Client, error)`

Crea un nuovo client Warden.

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

Recupera l'intero elenco degli utenti. Se la cache è valida, restituisce direttamente i dati memorizzati.

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

Recupera l'elenco paginato degli utenti.

- `page`: numero di pagina (a partire da 1)
- `pageSize`: dimensione della pagina

Restituisce `PaginatedResponse`, contenente:
- `Data`: elenco degli utenti
- `Pagination`: informazioni di paginazione (numero di pagina, dimensione, totale, numero di pagine)

**Nota:** questo metodo non usa la cache; ogni chiamata recupera i dati più recenti dall'API.

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

Recupera le informazioni di un singolo utente a partire da un identificatore.

- `phone`: numero di telefono dell'utente (facoltativo, ma occorre fornire uno tra phone, mail o userID)
- `mail`: e-mail dell'utente (facoltativa)
- `userID`: identificatore univoco dell'utente (facoltativo)

**Importante:** occorre fornire esattamente uno tra gli identificatori `phone`, `mail` o `userID`.

Restituisce `*AllowListUser` e un errore. Se l'utente non esiste, restituisce l'errore `ErrCodeNotFound`.

**Nota:** questo metodo non usa la cache; ogni chiamata recupera i dati più recenti dall'API.

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

Verifica se un utente è presente nella lista di autorizzazione.

- `phone`: numero di telefono dell'utente (facoltativo)
- `mail`: e-mail dell'utente (facoltativa)

Restituisce `true` se l'utente esiste (trovato tramite telefono o e-mail), altrimenti `false`.

**Comportamento:**
- Se vengono forniti sia `phone` sia `mail`, ha la precedenza `phone`
- Se la ricerca tramite `phone` fallisce (errore `NotFound`) e `mail` non è vuoto, si passa automaticamente alla ricerca tramite `mail`
- Se la ricerca tramite `phone` riesce ma lo stato dell'utente non è attivo, non si ripiega su `mail` (l'utente è già stato trovato)
- Se la ricerca tramite `phone` fallisce con un errore diverso da `NotFound` (ad es. un errore di rete), non si ripiega su `mail`
- L'input viene normalizzato automaticamente: `phone` viene ripulito dagli spazi, `mail` viene ripulita e convertita in minuscolo
- Questo metodo usa `GetUserByIdentifier` per la ricerca, più efficiente rispetto a scorrere l'elenco degli utenti
- Solo gli utenti con stato "active" restituiscono `true`

#### `ClearCache()`

Svuota la cache interna del client.

#### `InvalidateCache()`

Alias di `ClearCache()`, per coerenza con l'invalidazione guidata dagli eventi.

#### `Close()`

Arresta le goroutine in background (ad es. il listener di invalidazione della cache) e libera le risorse.
Va chiamato quando il client non serve più.

## Definizioni dei tipi

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // Numero di telefono dell'utente
    Mail   string   `json:"mail"`    // Indirizzo e-mail dell'utente
    UserID string   `json:"user_id"` // Identificatore univoco dell'utente (facoltativo, generato automaticamente)
    Status string   `json:"status"`  // Stato dell'utente (ad es. "active", "inactive", "suspended")
    Scope  []string `json:"scope"`   // Ambito di autorizzazione dell'utente (facoltativo)
    Role   string   `json:"role"`    // Ruolo dell'utente (facoltativo)
}
```

**Metodi:**
- `IsActive() bool`: verifica se lo stato dell'utente è "active"
- `IsValid() bool`: verifica se lo stato dell'utente è valido (attualmente è supportato solo "active")

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // Numero di pagina corrente (a partire da 1)
    PageSize   int `json:"page_size"`   // Dimensione della pagina
    Total      int `json:"total"`       // Numero totale di record
    TotalPages int `json:"total_pages"` // Numero totale di pagine
}
```

## Gestione degli errori

L'SDK usa tipi di errore personalizzati con codici di errore e informazioni dettagliate:

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // Gestire l'errore di autenticazione
        case warden.ErrCodeRequestFailed:
            // Gestire il fallimento della richiesta
        case warden.ErrCodeNotFound:
            // Gestire l'errore "non trovato"
        case warden.ErrCodeServerError:
            // Gestire l'errore del server
        // ...
        }
    }
}
```

### Codici di errore

- `ErrCodeInvalidConfig`: configurazione non valida
- `ErrCodeRequestFailed`: richiesta fallita
- `ErrCodeInvalidResponse`: formato di risposta non valido
- `ErrCodeUnauthorized`: non autorizzato
- `ErrCodeNotFound`: non trovato
- `ErrCodeServerError`: errore del server

## Buone pratiche

1. **Riutilizzare il client**: creare il client una sola volta e riutilizzarlo per tutto il ciclo di vita dell'applicazione
2. **Impostare un TTL della cache adeguato**: definire una durata della cache adatta alla frequenza di aggiornamento dei dati
3. **Usare Context**: passare un contesto per supportare annullamento e controllo del timeout
4. **Gestione degli errori**: controllare e gestire sempre gli errori
5. **Log**: in produzione usare un'implementazione di logger adeguata
6. **Chiudere il client**: chiamare `Close()` quando il client non serve più, per arrestare le goroutine in background
7. **Configurare i tentativi ripetuti**: abilitarli in produzione per assorbire i guasti transitori
8. **Trasporto personalizzato**: usare un trasporto personalizzato per scenari avanzati (TLS, proxy, pool di connessioni, ecc.)

## Documentazione di progettazione

### Principi di progettazione

1. **Semplice e immediato**: offre interfacce API essenziali
2. **Prestazioni elevate**: la cache integrata riduce le chiamate all'API
3. **Sicuro rispetto ai thread**: tutti i metodi sono sicuri in concorrenza
4. **Configurazione flessibile**: timeout, cache, logger e altro sono personalizzabili

### Progettazione dell'architettura

#### Componenti principali

1. **Client**: wrapper del client HTTP
2. **Cache**: cache in memoria sicura rispetto ai thread
3. **Options**: opzioni di configurazione (modello Builder)
4. **Logger**: interfaccia del logger (supporta diverse librerie di logging)

#### Sicurezza in concorrenza

- `http.Client` è sicuro in concorrenza
- `Cache` usa `sync.RWMutex` per garantire la sicurezza rispetto ai thread
- Tutti i campi di `Client` sono in sola lettura dopo la creazione
- Tutti i metodi sono sicuri rispetto ai thread e possono essere chiamati in concorrenza da più goroutine

#### Strategia di cache

1. **GetUsers()**: usa la cache
   - Controlla prima la cache
   - Se la cache è valida, restituisce direttamente
   - Se la cache non è valida o non esiste, recupera i dati dall'API e aggiorna la cache

2. **GetUsersPaginated()**: non usa la cache
   - Motivo: parametri di paginazione diversi producono risultati diversi
   - Mettere in cache per parametri di paginazione sarebbe complesso
   - Progettazione attuale: recupera i dati dall'API a ogni chiamata per garantirne l'accuratezza

3. **GetUserByIdentifier()**: non usa la cache
   - Motivo: servono le informazioni più aggiornate di un singolo utente per garantire la freschezza dei dati
   - Ogni chiamata interroga l'API per evitare incoerenze dovute alla cache

4. **CheckUserInList()**: non usa la cache
   - Usa `GetUserByIdentifier()` per interrogare direttamente un singolo utente
   - Ogni chiamata invia una richiesta all'API per garantire la freschezza dei dati
   - Supporta un fallback intelligente: quando la ricerca tramite telefono fallisce (NotFound) e mail non è vuota, passa automaticamente alla ricerca tramite mail
   - Ottimizzazione delle prestazioni: interrogare direttamente un singolo utente è più efficiente che scorrere l'intero elenco

#### Strategia di implementazione di CheckUserInList

Il metodo `CheckUserInList()` applica la strategia seguente:

1. **Normalizzazione dell'input**: rimuove automaticamente gli spazi iniziali e finali da phone e mail e converte mail in minuscolo
2. **Strategia di precedenza**: se vengono forniti sia phone sia mail, ha la precedenza phone
3. **Fallback intelligente**:
   - Quando la ricerca tramite phone restituisce l'errore `NotFound`, se mail non è vuota si passa automaticamente alla ricerca tramite mail
   - Quando la ricerca tramite phone riesce ma lo stato dell'utente non è attivo, non si ripiega su mail (l'utente è già stato trovato)
   - Quando la ricerca tramite phone incontra altri errori (ad es. di rete), non si ripiega su mail
4. **Verifica dello stato**: solo gli utenti con stato "active" restituiscono `true`
5. **Ottimizzazione delle prestazioni**: usa `GetUserByIdentifier()` per l'interrogazione diretta, evitando di recuperare l'intero elenco degli utenti

### RetryOptions

La struttura `RetryOptions` configura il comportamento dei tentativi ripetuti:

- `MaxRetries`: numero massimo di tentativi (predefinito 0, nessun tentativo)
- `RetryDelay`: attesa iniziale tra i tentativi (predefinita 100 ms)
- `MaxRetryDelay`: attesa massima tra i tentativi (predefinita 5 s)
- `BackoffMultiplier`: moltiplicatore del backoff esponenziale (predefinito 2.0)
- `RetryableStatusCodes`: codici di stato HTTP che attivano un nuovo tentativo (predefinito: 5xx)

**Nota:** gli errori di rete sono sempre ripetibili. Gli errori del client (4xx) non vengono mai ripetuti.

### Limitazioni note

1. **Cache della paginazione**: `GetUsersPaginated()` non usa la cache
   - È una scelta deliberata per garantire l'accuratezza dei dati
   - Se serve una cache per la paginazione, si possono adottare strategie più complesse

2. **Cache delle query su singolo utente**: `GetUserByIdentifier()` e `CheckUserInList()` non usano la cache
   - È una scelta deliberata per garantire la freschezza dei dati
   - Se serve una cache, si possono adottare strategie basate sugli identificatori utente

### Miglioramenti futuri

1. Supporto per middleware di richiesta/risposta
2. Supporto per la raccolta di metriche
3. Supporto per la configurazione del pool di connessioni
4. Supporto per il modello circuit breaker

## Esempio completo

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Creare il client
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)

    client, err := warden.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Recuperare tutti gli utenti
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // Recuperare un singolo utente tramite telefono
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            fmt.Println("User not found")
        } else {
            log.Fatal(err)
        }
    } else {
        fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
    }

    // Query paginata
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // Verificare l'utente
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // Svuotare la cache
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## Documentazione correlata

- [Documentazione dell'API](API.md) – Scopri i dettagli degli endpoint dell'API
- [Documentazione di configurazione](CONFIGURATION.md) – Scopri le opzioni di configurazione del server
