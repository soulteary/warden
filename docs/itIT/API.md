# Documentazione dell'API

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](../frFR/API.md) | [Italiano](API.md) | [日本語](../jaJP/API.md) | [Deutsch](../deDE/API.md) | [한국어](../koKR/API.md)

Questo documento fornisce informazioni dettagliate su tutti gli endpoint API offerti da Warden.

## Documentazione OpenAPI

Il progetto fornisce una specifica OpenAPI 3.0 completa nel file `openapi.yaml`.

È possibile utilizzare i seguenti strumenti per consultare e testare l'API:

1. **Swagger UI**: aprire il file `openapi.yaml` con [Swagger Editor](https://editor.swagger.io/)
2. **Postman**: importare il file `openapi.yaml` in Postman
3. **Redoc**: utilizzare Redoc per generare una pagina di documentazione dell'API ben curata

## Autenticazione

Alcuni endpoint richiedono l'autenticazione tramite chiave API. È possibile fornire le informazioni di autenticazione in due modi:

1. **Intestazione X-API-Key**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Intestazione Authorization Bearer**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

La chiave API si configura tramite la variabile d'ambiente `API_KEY` o l'argomento da riga di comando `--api-key`.

## Contratto di routing

Gli endpoint documentati di seguito costituiscono l'insieme completo dei percorsi serviti da
Warden. Qualsiasi altro percorso restituisce `404 Not Found` con un corpo JSON e senza alcun
dato utente:

```http
GET /not-a-route
X-API-Key: your-secret-api-key
```

```json
{
  "error": "Requested resource does not exist"
}
```

> **Cambiamento di comportamento**: il percorso radice `/` era in precedenza registrato come
> pattern di sottoalbero, per cui ogni percorso non corrispondente (`/foo`, `/user/`, `/v1/`)
> veniva servito dal gestore dell'elenco utenti e restituiva l'**intera lista di
> autorizzazione**. Ora `/` è una corrispondenza esatta e i percorsi non corrispondenti
> ricevono il 404 mostrato sopra. I client che facevano affidamento su un percorso arbitrario
> per ottenere dati utente devono usare `/`, `/data.json` oppure `/v1/users`.

Si noti che il router di Go normalizza il percorso prima della corrispondenza: `/metrics/../user`
viene quindi risolto in `/user` e reindirizzato lì, invece di raggiungere il gestore 404.

## Endpoint dell'API

### Ottenere l'elenco degli utenti

Ottenere tutti gli utenti o un elenco paginato.

**Richiesta**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**Parametri di query**:
- `page` (facoltativo): numero di pagina, a partire da 1, valore predefinito 1
- `page_size` (facoltativo): numero di elementi per pagina, per impostazione predefinita tutti i dati (nessuna paginazione)

**Nota**: questo endpoint richiede l'autenticazione tramite chiave API.

**Risposta (senza paginazione)**
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
```

**Risposta (con paginazione)**
```json
{
    "data": [
        {
            "phone": "13800138000",
            "mail": "admin@example.com"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 100,
        "total": 200,
        "total_pages": 2
    }
}
```

**Codice di stato**: `200 OK`

**Content-Type**: `application/json`

### Ottenere un singolo utente

Interrogare un singolo utente tramite numero di telefono, indirizzo e-mail o ID utente.

**Richiesta**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**Parametri di query** (deve esserne fornito esattamente uno):
- `phone`: numero di telefono dell'utente
- `mail`: indirizzo e-mail dell'utente
- `user_id`: identificatore univoco dell'utente

**Nota**:
- Questo endpoint richiede l'autenticazione tramite chiave API
- È consentito un solo parametro di query (`phone`, `mail` o `user_id`)

**Risposta (l'utente esiste)**
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**Descrizione dei campi**:
- `phone`: numero di telefono dell'utente
- `mail`: indirizzo e-mail dell'utente
- `user_id`: identificatore univoco dell'utente (generato automaticamente se non fornito)
- `status`: stato dell'utente, valori possibili:
  - `"active"`: attivo, l'utente può accedere e utilizzare il sistema
  - `"inactive"`: inattivo, l'utente non può accedere
  - `"suspended"`: sospeso, l'utente non può accedere
  - Il valore predefinito è `"inactive"` se non impostato; `"active"` deve essere impostato esplicitamente per consentire l'accesso
- `scope`: array degli ambiti di autorizzazione dell'utente (facoltativo), per un'autorizzazione granulare, ad esempio `["read", "write", "admin"]`
- `role`: ruolo dell'utente (facoltativo), ad esempio `"admin"`, `"user"`, `"guest"`

**Note**:
- Solo gli utenti con `status` pari a `"active"` superano i controlli di autenticazione
- I campi `scope` e `role` sono usati da Stargate per impostare le intestazioni di autorizzazione (`X-Auth-Scopes` e `X-Auth-Role`) destinate ai servizi a valle

**Scenario di integrazione facoltativo**:
Se si sceglie di integrarsi con altri servizi (come Stargate), è possibile chiamare questo endpoint per interrogare le informazioni utente nel flusso di accesso:
1. Dopo che l'utente ha inserito un identificatore (e-mail/telefono/nome utente), chiamare `GET /user?phone=xxx` oppure `GET /user?mail=xxx`
2. Warden restituisce le informazioni utente (inclusi `user_id`, `mail`, `phone`, `status`)
3. Se l'utente esiste e lo stato è `"active"`, è possibile proseguire con il flusso di autenticazione
4. I valori `scope` e `role` restituiti possono essere usati per impostare le intestazioni di autorizzazione

**Risposta (utente non trovato)**
- **Codice di stato**: `404 Not Found`
- **Corpo della risposta**: `User not found`

**Risposta di errore (parametro mancante)**
- **Codice di stato**: `400 Bad Request`
- **Corpo della risposta**: `Bad Request: missing identifier (phone, mail, or user_id)`

**Risposta di errore (parametri multipli)**
- **Codice di stato**: `400 Bad Request`
- **Corpo della risposta**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### Controllo di integrità

Verifica Redis, la cache dei dati, la provenienza dello snapshot e la sua freschezza.

**Richiesta**
```http
GET /health
GET /healthcheck
```

**Nota**: questo endpoint non richiede autenticazione, ma gli indirizzi IP di accesso possono essere limitati tramite la variabile d'ambiente `HEALTH_CHECK_IP_WHITELIST`. In produzione le risposte nascondono i singoli controlli.

**Risposta**
```json
{
    "status": "ok",
    "service": "warden",
    "checks": {
        "redis": {
            "name": "redis",
            "status": "ok",
            "latency_ms": 1,
            "timestamp": "2026-08-31T00:00:00Z"
        },
        "snapshot": {
            "name": "snapshot",
            "status": "ok",
            "latency_ms": 0,
            "timestamp": "2026-08-31T00:00:00Z",
            "metadata": {
                "source": "merged",
                "version": "a1b2c3d4",
                "age_seconds": 2.5
            }
        }
    },
    "timestamp": "2026-08-31T00:00:00Z",
    "total_latency_ms": 1
}
```

Risposta in produzione:

```json
{"status":"ok","service":"warden"}
```

**Codici di stato**:

- `200 OK`: lo stato aggregato è `ok` oppure `degraded`; `degraded` significa che il servizio è ancora funzionante.
- `503 Service Unavailable`: un controllo critico è fallito.
- `403 Forbidden`: il client è al di fuori di `HEALTH_CHECK_IP_WHITELIST`.

**Descrizione dei campi di risposta**:
- `status`: `ok`, `degraded` oppure `unhealthy`.
- `service`: nome del servizio (`warden`).
- `checks`: mappa presente solo in sviluppo/test con i risultati di `redis`, `data`, `snapshot` e `snapshot_freshness`.
- `checks.snapshot.metadata`: origine, versione ed età a bassa cardinalità, oltre a codici di motivo stabili per gli aggiornamenti; errori remoti grezzi, URL e credenziali non vengono mai esposti.
- `timestamp`, `total_latency_ms`: campi di temporizzazione aggregata presenti solo in sviluppo/test.

In `REMOTE_FIRST` e `ONLY_REMOTE`, `snapshot_freshness` è critico. Una provenienza
sconosciuta o un'età superiore a `SNAPSHOT_MAX_AGE` restituisce 503. Le modalità tolleranti
possono servire uno snapshot locale convalidato o l'ultimo snapshot valido noto come
`degraded` con HTTP 200.

### Gestione del livello di log

Ottenere e impostare dinamicamente i livelli di log.

#### Ottenere il livello di log corrente

**Richiesta**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**Risposta**
```json
{
    "level": "info"
}
```

**Nota**: questo endpoint richiede l'autenticazione tramite chiave API.

#### Impostare il livello di log

**Richiesta**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**Corpo della richiesta**:
```json
{
    "level": "debug"
}
```

**Livelli di log supportati**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**Risposta**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**Nota**:
- Questo endpoint richiede l'autenticazione tramite chiave API
- Tutte le operazioni di modifica del livello di log sono registrate nei log di audit di sicurezza

### Metriche Prometheus

Ottenere i dati delle metriche di monitoraggio in formato Prometheus.

**Richiesta**
```http
GET /metrics
```

**Risposta**: dati delle metriche in formato Prometheus

**Autenticazione**: dipende dall'ambiente di distribuzione.

| `ENVIRONMENT` | Impostazione predefinita per `/metrics` |
| --- | --- |
| `production` | Autenticazione richiesta (stessi schemi degli endpoint dei dati) |
| `development`, `test`, non impostato | Scraping anonimo consentito |

`WARDEN_METRICS_REQUIRE_AUTH` sovrascrive l'impostazione predefinita in entrambe le direzioni.
Uno scraping non autenticato di un endpoint che richiede autenticazione restituisce
`401 Unauthorized`. La risposta contiene esclusivamente serie a bassa cardinalità e non
sensibili: le etichette `endpoint` e `method` sono normalizzate rispetto a una lista di
autorizzazione e i valori non riconosciuti confluiscono in `other`.

**Esempio di risposta**:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 1234

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/",le="0.005"} 1000
http_request_duration_seconds_bucket{method="GET",path="/",le="0.01"} 1200
...
```

## Risposte di errore

Tutti gli endpoint possono restituire le seguenti risposte di errore:

### 401 Unauthorized

Restituita quando l'autenticazione tramite chiave API fallisce:

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

Restituita quando le richieste superano il limite di frequenza:

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

Restituita quando si verifica un errore interno del server:

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

In modalità produzione le informazioni dettagliate sugli errori vengono nascoste per evitare fughe di informazioni.

## Limitazione di frequenza

Per impostazione predefinita, le richieste all'API sono protette da una limitazione di frequenza:

- **Limite**: 60 richieste al minuto
- **Finestra**: 1 minuto
- **In caso di superamento**: restituisce `429 Too Many Requests`

La limitazione di frequenza può essere modificata tramite il file di configurazione:

```yaml
rate_limit:
  rate: 60  # Richieste al minuto
  window: 1m
```

## Lista di autorizzazione IP

Le liste di autorizzazione IP si configurano tramite le seguenti variabili d'ambiente:

- `IP_WHITELIST`: lista di autorizzazione globale (limita l'accesso a tutti gli endpoint)
- `HEALTH_CHECK_IP_WHITELIST`: lista di autorizzazione per l'endpoint di controllo integrità (limita solo `/health` e `/healthcheck`)

È supportato il formato di intervallo CIDR; più indirizzi IP o intervalli vanno separati da virgole:

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Compressione delle risposte

Tutte le risposte dell'API supportano la compressione automatica (gzip). I client possono attivare la compressione tramite l'intestazione di richiesta `Accept-Encoding: gzip`.

## Esempi di integrazione facoltativi

### Esempio di chiamata per l'integrazione con altri servizi (facoltativo)

Se è necessario integrarsi con altri servizi (come Stargate), è possibile chiamare l'endpoint `/user` di Warden per interrogare le informazioni utente nel flusso di accesso:

**Scenario 1: interrogazione tramite numero di telefono**

```bash
# Stargate chiama Warden
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?phone=13800138000"
```

**Esempio di risposta**:
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**Scenario 2: interrogazione tramite e-mail**

```bash
# Stargate chiama Warden
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?mail=admin@example.com"
```

### Esempio di integrazione con l'SDK Go

Stargate può utilizzare l'SDK Go di Warden per l'integrazione:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Creare il client Warden
    opts := warden.DefaultOptions().
        WithBaseURL("http://warden:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second)
    
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // Interrogare l'utente nel flusso di accesso
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            // Utente non trovato, rifiutare l'accesso
            fmt.Println("User not found in allowlist")
            return
        }
        panic(err)
    }
    
    // Verificare lo stato dell'utente
    if !user.IsActive() {
        // Lo stato dell'utente non è attivo, rifiutare l'accesso
        fmt.Printf("User status is %s, cannot login\n", user.Status)
        return
    }
    
    // L'utente esiste ed è attivo, proseguire con il flusso di accesso
    fmt.Printf("User found: %s, Status: %s, Role: %s, Scopes: %v\n",
        user.UserID, user.Status, user.Role, user.Scope)
    
    // Passo successivo: chiamare Herald per inviare un codice di verifica
    // ...
}
```

### Esempio di flusso di accesso completo (scenario di integrazione facoltativo)

Negli scenari di integrazione facoltativi, il flusso di accesso completo può presentarsi così:

1. **L'utente inserisce un identificatore** → il servizio di autenticazione lo riceve
2. **Servizio di autenticazione → Warden**: interrogare le informazioni utente
   ```go
   user, err := wardenClient.GetUserByIdentifier(ctx, phone, mail, "")
   ```
3. **Convalidare lo stato dell'utente**: verificare `user.Status == "active"`
4. **Servizio di autenticazione → servizio OTP**: creare una challenge e inviare il codice di verifica (facoltativo)
5. **L'utente invia il codice di verifica** → il servizio di autenticazione lo riceve (facoltativo)
6. **Servizio di autenticazione → servizio OTP**: verificare il codice (facoltativo)
7. **Servizio di autenticazione**: emettere una sessione e usare `user.Scope` e `user.Role` per impostare le intestazioni di autorizzazione

**Nota**: Warden può essere utilizzato in modo autonomo; il flusso di integrazione sopra descritto è facoltativo.

## Documentazione correlata

- [Specifica OpenAPI](../../openapi.yaml) – Specifica OpenAPI 3.1 completa
- [Documentazione di configurazione](CONFIGURATION.md) – Scopri come configurare la chiave API e le altre opzioni
- [Documentazione di sicurezza](SECURITY.md) – Scopri le funzionalità di sicurezza e le buone pratiche
- [Documentazione di architettura](ARCHITECTURE.md) – Scopri l'architettura di integrazione dei servizi
