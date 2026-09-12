# SDK-Nutzungsdokumentation

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](../zhCN/SDK.md) | [Français](../frFR/SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](SDK.md) | [한국어](../koKR/SDK.md)

Warden stellt ein Go-SDK bereit, um die Einbindung in andere Projekte zu erleichtern. Das SDK bietet eine schlanke API-Schnittstelle mit Unterstützung für Caching, Authentifizierung und mehr.

## Funktionen

- 🚀 **Einfach und unkompliziert**: Bietet schlanke API-Schnittstellen
- ⚡ **Hohe Leistung**: Eingebaute Cache-Unterstützung (GetUsers); direkte Abfragen (GetUserByIdentifier) reduzieren API-Aufrufe
- 🔒 **Sicher**: Unterstützt API-Key-Authentifizierung; die Fehlerbehandlung gibt keine vertraulichen Informationen preis
- 📦 **Flexibel**: Timeout, Cache-TTL und mehr sind konfigurierbar
- 🔌 **Erweiterbar**: Unterstützt eigene Logger-Implementierungen
- 🎯 **Intelligenter Fallback**: CheckUserInList fällt automatisch auf die E-Mail-Adresse zurück, wenn die Telefonnummer nicht gefunden wird

## SDK installieren

```bash
go get github.com/soulteary/warden/pkg/warden
```

## Schnellstart

### Grundlegende Verwendung

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Client-Optionen erstellen
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)
    
    // Client erstellen
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    // Benutzerliste abrufen
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }
    
    // Prüfen, ob der Benutzer in der Liste steht (phone, mail oder beides möglich)
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // Es lässt sich auch nur phone oder nur mail verwenden
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // Benutzerdetails abrufen
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### Eigenen Logger verwenden

Das SDK unterstützt eigene Logger-Implementierungen, zum Beispiel mit logrus:

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

### Seitenweise Abfrage

```go
// Seitenweise Benutzerliste abrufen
resp, err := client.GetUsersPaginated(ctx, 1, 10) // Seite 1, 10 Einträge pro Seite
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

### Informationen zu einem einzelnen Benutzer abrufen

```go
// Benutzerinformationen per Telefonnummer abrufen
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

// Benutzerinformationen per E-Mail-Adresse abrufen
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// Benutzerinformationen per Benutzer-ID abrufen
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
```

### Cache leeren

```go
// Client-Cache manuell leeren
client.ClearCache()

// Oder den Alias verwenden
client.InvalidateCache()
```

### Eigener HTTP-Transport

```go
import "net/http"

// Eigenen Transport erstellen
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### Signieren von Anfragen mit HMAC v2

```go
opts := warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET"))

client, err := warden.NewClient(opts)
```

Das SDK signiert Key-ID, Zeitstempel, Nonce, Pfad/Query, Methode und den Hash des Bodys.
Key-ID und Geheimnis müssen beide konfiguriert sein; ein unvollständiges Paar liefert
`ErrCodeInvalidConfig`, statt eine unsignierte Anfrage zu senden.

### Konfiguration von Wiederholungsversuchen

```go
// Optionen für Wiederholungsversuche konfigurieren
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

### Ereignisgesteuerte Cache-Invalidierung

```go
// Kanal für Ereignisse zur Cache-Invalidierung erstellen
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // Wichtig: schließen, um den Hintergrund-Listener zu beenden

// Später die Cache-Invalidierung durch ein externes Ereignis auslösen
invalidationCh <- struct{}{}

// Der Cache wird automatisch geleert, sobald das Signal empfangen wird
```

## API-Referenz

### Options

Die Struktur `Options` dient der Konfiguration des Clients:

- `BaseURL`: Adresse des Warden-Dienstes (erforderlich)
- `APIKey`: API-Key (optional)
- `Timeout`: Timeout für HTTP-Anfragen (Standard: 10 Sekunden)
- `CacheTTL`: Cache-TTL (Standard: 5 Minuten)
- `Logger`: Logger-Schnittstelle (optional, standardmäßig NoOpLogger)
- `Transport`: Eigener HTTP-Transport (optional)
- `HMACKeyID` / `HMACSecret`: Signaturpaar für HMAC v2 (beide oder keines)
- `TLSConfig`: TLS-/mTLS-Clientkonfiguration (optional)
- `Retry`: Konfiguration der Wiederholungsversuche (optional, standardmäßig keine Wiederholung)
- `CacheInvalidationChannel`: Kanal für ereignisgesteuerte Cache-Invalidierung (optional)

### Client-Methoden

#### `NewClient(opts *Options) (*Client, error)`

Erstellt einen neuen Warden-Client.

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

Ruft die vollständige Benutzerliste ab. Ist der Cache gültig, werden direkt die zwischengespeicherten Daten zurückgegeben.

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

Ruft die Benutzerliste seitenweise ab.

- `page`: Seitenzahl (beginnt bei 1)
- `pageSize`: Seitengröße

Liefert `PaginatedResponse` mit:
- `Data`: Benutzerliste
- `Pagination`: Informationen zur Seitenaufteilung (Seitenzahl, Seitengröße, Gesamtzahl, Gesamtseiten)

**Hinweis:** Diese Methode nutzt keinen Cache; jeder Aufruf holt die aktuellen Daten von der API.

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

Ruft die Informationen zu einem einzelnen Benutzer anhand einer Kennung ab.

- `phone`: Telefonnummer des Benutzers (optional, aber eines von phone, mail oder userID muss angegeben werden)
- `mail`: E-Mail-Adresse des Benutzers (optional)
- `userID`: Eindeutige Kennung des Benutzers (optional)

**Wichtig:** Es muss genau eine der Kennungen `phone`, `mail` oder `userID` angegeben werden.

Liefert `*AllowListUser` und einen Fehler. Existiert der Benutzer nicht, wird der Fehler `ErrCodeNotFound` zurückgegeben.

**Hinweis:** Diese Methode nutzt keinen Cache; jeder Aufruf holt die aktuellen Daten von der API.

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

Prüft, ob ein Benutzer in der Allow-Liste steht.

- `phone`: Telefonnummer des Benutzers (optional)
- `mail`: E-Mail-Adresse des Benutzers (optional)

Liefert `true`, wenn der Benutzer existiert (gefunden über Telefonnummer oder E-Mail-Adresse), sonst `false`.

**Verhalten:**
- Sind sowohl `phone` als auch `mail` angegeben, hat `phone` Vorrang
- Schlägt die Suche über `phone` fehl (Fehler `NotFound`) und ist `mail` nicht leer, wird automatisch auf die Suche über `mail` zurückgegriffen
- Ist die Suche über `phone` erfolgreich, der Benutzerstatus aber nicht aktiv, erfolgt kein Rückgriff auf `mail` (der Benutzer wurde bereits gefunden)
- Schlägt die Suche über `phone` fehl und ist der Fehler nicht `NotFound` (etwa ein Netzwerkfehler), erfolgt kein Rückgriff auf `mail`
- Die Eingabe wird automatisch normalisiert: `phone` wird getrimmt, `mail` wird getrimmt und in Kleinbuchstaben umgewandelt
- Diese Methode nutzt `GetUserByIdentifier` für die Suche, was effizienter ist als das Durchlaufen der Benutzerliste
- Nur Benutzer mit dem Status „active“ liefern `true`

#### `ClearCache()`

Leert den internen Cache des Clients.

#### `InvalidateCache()`

Alias für `ClearCache()`, zur Konsistenz mit der ereignisgesteuerten Invalidierung.

#### `Close()`

Beendet Hintergrund-Goroutinen (etwa den Listener für die Cache-Invalidierung) und gibt Ressourcen frei.
Sollte aufgerufen werden, sobald der Client nicht mehr benötigt wird.

## Typdefinitionen

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // Telefonnummer des Benutzers
    Mail   string   `json:"mail"`    // E-Mail-Adresse des Benutzers
    UserID string   `json:"user_id"` // Eindeutige Kennung des Benutzers (optional, wird automatisch erzeugt)
    Status string   `json:"status"`  // Benutzerstatus (z. B. "active", "inactive", "suspended")
    Scope  []string `json:"scope"`   // Berechtigungsbereich des Benutzers (optional)
    Role   string   `json:"role"`    // Rolle des Benutzers (optional)
}
```

**Methoden:**
- `IsActive() bool`: Prüft, ob der Benutzerstatus „active“ ist
- `IsValid() bool`: Prüft, ob der Benutzerstatus gültig ist (derzeit wird nur „active“ unterstützt)

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // Aktuelle Seitenzahl (beginnt bei 1)
    PageSize   int `json:"page_size"`   // Seitengröße
    Total      int `json:"total"`       // Gesamtzahl der Datensätze
    TotalPages int `json:"total_pages"` // Gesamtzahl der Seiten
}
```

## Fehlerbehandlung

Das SDK verwendet eigene Fehlertypen mit Fehlercodes und Detailinformationen:

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // Authentifizierungsfehler behandeln
        case warden.ErrCodeRequestFailed:
            // Fehlgeschlagene Anfrage behandeln
        case warden.ErrCodeNotFound:
            // Fehler "nicht gefunden" behandeln
        case warden.ErrCodeServerError:
            // Serverfehler behandeln
        // ...
        }
    }
}
```

### Fehlercodes

- `ErrCodeInvalidConfig`: Ungültige Konfiguration
- `ErrCodeRequestFailed`: Anfrage fehlgeschlagen
- `ErrCodeInvalidResponse`: Ungültiges Antwortformat
- `ErrCodeUnauthorized`: Nicht autorisiert
- `ErrCodeNotFound`: Nicht gefunden
- `ErrCodeServerError`: Serverfehler

## Best Practices

1. **Client wiederverwenden**: Erstellen Sie den Client einmal und verwenden Sie ihn über die gesamte Lebensdauer der Anwendung
2. **Cache-TTL passend wählen**: Legen Sie die Cache-Dauer entsprechend der Aktualisierungsfrequenz der Daten fest
3. **Context verwenden**: Übergeben Sie einen Context, um Abbruch und Timeout steuern zu können
4. **Fehlerbehandlung**: Prüfen und behandeln Sie Fehler stets
5. **Logging**: Verwenden Sie in Produktionsumgebungen eine geeignete Logger-Implementierung
6. **Client schließen**: Rufen Sie `Close()` auf, sobald der Client nicht mehr benötigt wird, um Hintergrund-Goroutinen zu beenden
7. **Wiederholungsversuche konfigurieren**: Aktivieren Sie Wiederholungen in der Produktion, um vorübergehende Ausfälle abzufangen
8. **Eigener Transport**: Verwenden Sie einen eigenen Transport für fortgeschrittene Szenarien (TLS, Proxy, Verbindungspooling usw.)

## Entwurfsdokumentation

### Entwurfsprinzipien

1. **Einfach und unkompliziert**: Bietet schlanke API-Schnittstellen
2. **Hohe Leistung**: Eingebaute Cache-Unterstützung reduziert API-Aufrufe
3. **Thread-sicher**: Alle Methoden sind nebenläufigkeitssicher
4. **Flexible Konfiguration**: Unterstützt eigene Timeouts, Caches, Logger und mehr

### Architekturentwurf

#### Kernkomponenten

1. **Client**: Wrapper um den HTTP-Client
2. **Cache**: Thread-sicherer In-Memory-Cache
3. **Options**: Konfigurationsoptionen (Builder-Muster)
4. **Logger**: Logger-Schnittstelle (unterstützt verschiedene Logging-Bibliotheken)

#### Nebenläufigkeitssicherheit

- `http.Client` ist nebenläufigkeitssicher
- `Cache` verwendet `sync.RWMutex`, um Thread-Sicherheit zu gewährleisten
- Alle Felder von `Client` sind nach der Erstellung schreibgeschützt
- Alle Methoden sind thread-sicher und können nebenläufig aus mehreren Goroutinen aufgerufen werden

#### Cache-Strategie

1. **GetUsers()**: Verwendet den Cache
   - Prüft zuerst den Cache
   - Ist der Cache gültig, wird direkt zurückgegeben
   - Ist der Cache ungültig oder nicht vorhanden, werden die Daten von der API geholt und der Cache aktualisiert

2. **GetUsersPaginated()**: Verwendet keinen Cache
   - Grund: Unterschiedliche Seitenparameter liefern unterschiedliche Ergebnisse
   - Ein Cache pro Seitenparameterkombination wäre komplex
   - Aktueller Entwurf: Holt die Daten jedes Mal von der API, um korrekte Daten sicherzustellen

3. **GetUserByIdentifier()**: Verwendet keinen Cache
   - Grund: Die aktuellsten Informationen zu einem einzelnen Benutzer werden benötigt, um Aktualität sicherzustellen
   - Jeder Aufruf holt die Daten von der API, um Inkonsistenzen durch den Cache zu vermeiden

4. **CheckUserInList()**: Verwendet keinen Cache
   - Nutzt `GetUserByIdentifier()`, um direkt einen einzelnen Benutzer abzufragen
   - Jeder Aufruf sendet eine API-Anfrage, um Aktualität sicherzustellen
   - Unterstützt einen intelligenten Fallback: Schlägt die Suche über die Telefonnummer fehl (NotFound) und ist mail nicht leer, wird automatisch über mail gesucht
   - Leistungsoptimierung: Die direkte Abfrage eines einzelnen Benutzers ist effizienter als das Durchlaufen der gesamten Benutzerliste

#### Implementierungsstrategie von CheckUserInList

Die Methode `CheckUserInList()` verwendet folgende Strategie:

1. **Normalisierung der Eingabe**: Führende und abschließende Leerzeichen werden bei phone und mail automatisch entfernt, mail wird in Kleinbuchstaben umgewandelt
2. **Prioritätsstrategie**: Sind sowohl phone als auch mail angegeben, hat phone Vorrang
3. **Intelligenter Fallback**:
   - Liefert die Suche über phone den Fehler `NotFound` und ist mail nicht leer, wird automatisch über mail gesucht
   - Ist die Suche über phone erfolgreich, der Benutzerstatus aber nicht aktiv, erfolgt kein Rückgriff auf mail (der Benutzer wurde bereits gefunden)
   - Tritt bei der Suche über phone ein anderer Fehler auf (etwa ein Netzwerkfehler), erfolgt kein Rückgriff auf mail
4. **Statusprüfung**: Nur Benutzer mit dem Status „active“ liefern `true`
5. **Leistungsoptimierung**: Nutzt `GetUserByIdentifier()` für die direkte Abfrage, statt die gesamte Benutzerliste zu holen

### RetryOptions

Die Struktur `RetryOptions` konfiguriert das Verhalten bei Wiederholungsversuchen:

- `MaxRetries`: Maximale Anzahl an Wiederholungen (Standard 0, keine Wiederholung)
- `RetryDelay`: Anfängliche Wartezeit zwischen Wiederholungen (Standard 100 ms)
- `MaxRetryDelay`: Maximale Wartezeit zwischen Wiederholungen (Standard 5 s)
- `BackoffMultiplier`: Faktor für den exponentiellen Backoff (Standard 2.0)
- `RetryableStatusCodes`: HTTP-Statuscodes, die eine Wiederholung auslösen (Standard: 5xx)

**Hinweis:** Netzwerkfehler sind stets wiederholbar. Clientfehler (4xx) werden niemals wiederholt.

### Bekannte Einschränkungen

1. **Cache für seitenweise Abfragen**: `GetUsersPaginated()` verwendet keinen Cache
   - Das ist bewusst so entworfen, um korrekte Daten sicherzustellen
   - Wird ein Cache für Seitenabfragen benötigt, lassen sich komplexere Strategien umsetzen

2. **Cache für Einzelbenutzerabfragen**: `GetUserByIdentifier()` und `CheckUserInList()` verwenden keinen Cache
   - Das ist bewusst so entworfen, um Aktualität sicherzustellen
   - Wird Caching benötigt, lassen sich Strategien auf Basis von Benutzerkennungen umsetzen

### Künftige Verbesserungen

1. Unterstützung von Middleware für Anfrage/Antwort
2. Unterstützung für das Sammeln von Metriken
3. Unterstützung für die Konfiguration von Verbindungspools
4. Unterstützung des Circuit-Breaker-Musters

## Vollständiges Beispiel

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
    // Client erstellen
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

    // Alle Benutzer abrufen
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // Einzelnen Benutzer per Telefonnummer abrufen
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

    // Seitenweise Abfrage
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // Benutzer prüfen
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // Cache leeren
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## Verwandte Dokumentation

- [API-Dokumentation](API.md) – Erfahren Sie mehr über die Details der API-Endpunkte
- [Konfigurationsdokumentation](CONFIGURATION.md) – Erfahren Sie mehr über die Konfigurationsoptionen des Servers
