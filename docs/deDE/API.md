# API-Dokumentation

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](../frFR/API.md) | [Italiano](../itIT/API.md) | [日本語](../jaJP/API.md) | [Deutsch](API.md) | [한국어](../koKR/API.md)

Dieses Dokument enthält detaillierte Informationen zu allen von Warden bereitgestellten API-Endpunkten.

## OpenAPI-Dokumentation

Das Projekt stellt in der Datei `openapi.yaml` eine vollständige Spezifikation nach OpenAPI 3.0 bereit.

Sie können die folgenden Werkzeuge verwenden, um die API anzusehen und zu testen:

1. **Swagger UI**: Öffnen Sie die Datei `openapi.yaml` mit dem [Swagger Editor](https://editor.swagger.io/)
2. **Postman**: Importieren Sie die Datei `openapi.yaml` in Postman
3. **Redoc**: Verwenden Sie Redoc, um eine ansprechende API-Dokumentationsseite zu erzeugen

## Authentifizierung

Einige API-Endpunkte erfordern eine Authentifizierung per API-Key. Sie können die Authentifizierungsinformationen auf zwei Arten übergeben:

1. **X-API-Key-Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization-Bearer-Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

Der API-Key kann über die Umgebungsvariable `API_KEY` oder das Kommandozeilenargument `--api-key` konfiguriert werden.

## Routing-Vertrag

Die unten dokumentierten Endpunkte sind die vollständige Menge der von Warden bedienten
Pfade. Jeder andere Pfad liefert `404 Not Found` mit einem JSON-Body und ohne Benutzerdaten:

```http
GET /not-a-route
X-API-Key: your-secret-api-key
```

```json
{
  "error": "Requested resource does not exist"
}
```

> **Verhaltensänderung**: Der Wurzelpfad `/` war früher als Teilbaum-Muster registriert,
> sodass jeder nicht zugeordnete Pfad (`/foo`, `/user/`, `/v1/`) vom Handler für die
> Benutzerliste bedient wurde und die **vollständige Allow-Liste** zurückgab. `/` ist
> jetzt eine exakte Übereinstimmung, und nicht zugeordnete Pfade erhalten den obigen 404.
> Clients, die sich darauf verlassen haben, dass ein beliebiger Pfad Benutzerdaten liefert,
> müssen `/`, `/data.json` oder `/v1/users` verwenden.

Beachten Sie, dass der Router von Go den Pfad vor dem Abgleich bereinigt: `/metrics/../user`
wird also zu `/user` aufgelöst und dorthin weitergeleitet, statt den 404-Handler zu erreichen.

## API-Endpunkte

### Benutzerliste abrufen

Alle Benutzer oder eine seitenweise Benutzerliste abrufen.

**Anfrage**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**Query-Parameter**:
- `page` (optional): Seitenzahl, beginnend bei 1, Standardwert 1
- `page_size` (optional): Anzahl der Einträge pro Seite, standardmäßig alle Daten (keine Seitenaufteilung)

**Hinweis**: Dieser Endpunkt erfordert eine Authentifizierung per API-Key.

**Antwort (ohne Seitenaufteilung)**
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

**Antwort (mit Seitenaufteilung)**
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

**Statuscode**: `200 OK`

**Content-Type**: `application/json`

### Einzelnen Benutzer abrufen

Einen einzelnen Benutzer anhand von Telefonnummer, E-Mail-Adresse oder Benutzer-ID abfragen.

**Anfrage**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**Query-Parameter** (genau einer muss angegeben werden):
- `phone`: Telefonnummer des Benutzers
- `mail`: E-Mail-Adresse des Benutzers
- `user_id`: Eindeutige Kennung des Benutzers

**Hinweis**:
- Dieser Endpunkt erfordert eine Authentifizierung per API-Key
- Es ist nur ein Query-Parameter (`phone`, `mail` oder `user_id`) zulässig

**Antwort (Benutzer existiert)**
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

**Feldbeschreibungen**:
- `phone`: Telefonnummer des Benutzers
- `mail`: E-Mail-Adresse des Benutzers
- `user_id`: Eindeutige Kennung des Benutzers (wird automatisch erzeugt, wenn nicht angegeben)
- `status`: Benutzerstatus, mögliche Werte:
  - `"active"`: Aktiv, der Benutzer kann sich anmelden und auf das System zugreifen
  - `"inactive"`: Inaktiv, der Benutzer kann sich nicht anmelden
  - `"suspended"`: Gesperrt, der Benutzer kann sich nicht anmelden
  - Standardwert ist `"inactive"`, wenn nichts gesetzt ist; `"active"` muss explizit gesetzt werden, um die Anmeldung zu erlauben
- `scope`: Array der Berechtigungsbereiche des Benutzers (optional), für feingranulare Autorisierung, z. B. `["read", "write", "admin"]`
- `role`: Rolle des Benutzers (optional), z. B. `"admin"`, `"user"`, `"guest"`

**Hinweise**:
- Nur Benutzer mit dem `status` `"active"` bestehen die Authentifizierungsprüfungen
- Die Felder `scope` und `role` werden von Stargate verwendet, um Autorisierungs-Header (`X-Auth-Scopes` und `X-Auth-Role`) für nachgelagerte Dienste zu setzen

**Optionales Integrationsszenario**:
Wenn Sie eine Integration mit anderen Diensten (etwa Stargate) wählen, können Sie diesen Endpunkt aufrufen, um im Anmeldeablauf Benutzerinformationen abzufragen:
1. Nachdem der Benutzer eine Kennung eingegeben hat (E-Mail/Telefon/Benutzername), rufen Sie `GET /user?phone=xxx` oder `GET /user?mail=xxx` auf
2. Warden liefert die Benutzerinformationen zurück (einschließlich `user_id`, `email`, `phone`, `status`)
3. Wenn der Benutzer existiert und der Status `"active"` ist, können Sie den weiteren Authentifizierungsablauf fortsetzen
4. Die zurückgegebenen Werte `scope` und `role` können zum Setzen der Autorisierungs-Header verwendet werden

**Antwort (Benutzer nicht gefunden)**
- **Statuscode**: `404 Not Found`
- **Antwortkörper**: `User not found`

**Fehlerantwort (fehlender Parameter)**
- **Statuscode**: `400 Bad Request`
- **Antwortkörper**: `Bad Request: missing identifier (phone, mail, or user_id)`

**Fehlerantwort (mehrere Parameter)**
- **Statuscode**: `400 Bad Request`
- **Antwortkörper**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### Health-Check

Prüft Redis, den Daten-Cache, die Herkunft des Snapshots und dessen Aktualität.

**Anfrage**
```http
GET /health
GET /healthcheck
```

**Hinweis**: Dieser Endpunkt erfordert keine Authentifizierung, die zugreifenden IP-Adressen können jedoch über die Umgebungsvariable `HEALTH_CHECK_IP_WHITELIST` eingeschränkt werden. In der Produktion verbergen die Antworten die einzelnen Prüfungen.

**Antwort**
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

Antwort in der Produktion:

```json
{"status":"ok","service":"warden"}
```

**Statuscodes**:

- `200 OK`: Der Gesamtstatus ist `ok` oder `degraded`; `degraded` bedeutet, dass der Dienst weiterhin funktionsfähig ist.
- `503 Service Unavailable`: Eine kritische Prüfung ist fehlgeschlagen.
- `403 Forbidden`: Der Client liegt außerhalb von `HEALTH_CHECK_IP_WHITELIST`.

**Beschreibung der Antwortfelder**:
- `status`: `ok`, `degraded` oder `unhealthy`.
- `service`: Name des Dienstes (`warden`).
- `checks`: Nur in Entwicklung/Test vorhandene Map mit den Ergebnissen von `redis`, `data`, `snapshot` und `snapshot_freshness`.
- `checks.snapshot.metadata`: Quelle, Version und Alter mit geringer Kardinalität sowie stabile Ursachencodes für Aktualisierungen; rohe Remote-Fehler, URLs und Zugangsdaten werden niemals offengelegt.
- `timestamp`, `total_latency_ms`: Nur in Entwicklung/Test vorhandene Felder zur Gesamtlaufzeit.

In `REMOTE_FIRST` und `ONLY_REMOTE` ist `snapshot_freshness` kritisch. Unbekannte
Herkunft oder ein Alter jenseits von `SNAPSHOT_MAX_AGE` führt zu 503. Tolerante Modi
können einen validierten lokalen oder zuletzt bekannten guten Snapshot als `degraded`
mit HTTP 200 ausliefern.

### Verwaltung des Log-Levels

Log-Level dynamisch abfragen und setzen.

#### Aktuelles Log-Level abfragen

**Anfrage**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**Antwort**
```json
{
    "level": "info"
}
```

**Hinweis**: Dieser Endpunkt erfordert eine Authentifizierung per API-Key.

#### Log-Level setzen

**Anfrage**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**Anfragekörper**:
```json
{
    "level": "debug"
}
```

**Unterstützte Log-Level**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**Antwort**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**Hinweis**:
- Dieser Endpunkt erfordert eine Authentifizierung per API-Key
- Alle Änderungen des Log-Levels werden in den Sicherheits-Audit-Logs festgehalten

### Prometheus-Metriken

Monitoring-Metriken im Prometheus-Format abrufen.

**Anfrage**
```http
GET /metrics
```

**Antwort**: Metrikdaten im Prometheus-Format

**Authentifizierung**: abhängig von der Bereitstellungsumgebung.

| `ENVIRONMENT` | Standard für `/metrics` |
| --- | --- |
| `production` | Authentifizierung erforderlich (dieselben Verfahren wie bei den Datenendpunkten) |
| `development`, `test`, nicht gesetzt | Anonymes Scraping erlaubt |

`WARDEN_METRICS_REQUIRE_AUTH` überschreibt den Standardwert in beide Richtungen. Ein nicht
authentifiziertes Scraping eines Endpunkts, der eine Authentifizierung erfordert, liefert
`401 Unauthorized`. Die Antwort enthält ausschließlich Zeitreihen mit geringer Kardinalität
und ohne sensible Daten: Die Labels `endpoint` und `method` werden gegen eine Allow-Liste
normalisiert, und nicht erkannte Werte fallen zu `other` zusammen.

**Beispielantwort**:
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

## Fehlerantworten

Alle API-Endpunkte können die folgenden Fehlerantworten liefern:

### 401 Unauthorized

Wird zurückgegeben, wenn die Authentifizierung per API-Key fehlschlägt:

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

Wird zurückgegeben, wenn die Anfragen die Ratenbegrenzung überschreiten:

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

Wird zurückgegeben, wenn ein interner Serverfehler auftritt:

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

Im Produktionsmodus werden detaillierte Fehlerinformationen verborgen, um Informationsabfluss zu verhindern.

## Ratenbegrenzung

Standardmäßig sind API-Anfragen durch eine Ratenbegrenzung geschützt:

- **Limit**: 60 Anfragen pro Minute
- **Zeitfenster**: 1 Minute
- **Bei Überschreitung**: Liefert `429 Too Many Requests`

Die Ratenbegrenzung lässt sich über die Konfigurationsdatei anpassen:

```yaml
rate_limit:
  rate: 60  # Anfragen pro Minute
  window: 1m
```

## IP-Allow-Liste

IP-Allow-Listen können über die folgenden Umgebungsvariablen konfiguriert werden:

- `IP_WHITELIST`: Globale IP-Allow-Liste (schränkt den Zugriff auf alle Endpunkte ein)
- `HEALTH_CHECK_IP_WHITELIST`: IP-Allow-Liste für den Health-Check-Endpunkt (schränkt nur `/health` und `/healthcheck` ein)

Das CIDR-Bereichsformat wird unterstützt; mehrere IP-Adressen oder Bereiche werden durch Kommas getrennt:

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Antwortkomprimierung

Alle API-Antworten unterstützen automatische Komprimierung (gzip). Clients können die Komprimierung über den Anfrage-Header `Accept-Encoding: gzip` aktivieren.

## Optionale Integrationsbeispiele

### Aufrufbeispiel für die Integration mit anderen Diensten (optional)

Wenn Sie eine Integration mit anderen Diensten (etwa Stargate) benötigen, können Sie im Anmeldeablauf den Endpunkt `/user` von Warden aufrufen, um Benutzerinformationen abzufragen:

**Szenario 1: Abfrage per Telefonnummer**

```bash
# Stargate ruft Warden auf
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?phone=13800138000"
```

**Beispielantwort**:
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

**Szenario 2: Abfrage per E-Mail**

```bash
# Stargate ruft Warden auf
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?mail=admin@example.com"
```

### Integrationsbeispiel mit dem Go-SDK

Stargate kann für die Integration das Warden-Go-SDK verwenden:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Warden-Client erstellen
    opts := warden.DefaultOptions().
        WithBaseURL("http://warden:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second)
    
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // Benutzer im Anmeldeablauf abfragen
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            // Benutzer nicht gefunden, Anmeldung ablehnen
            fmt.Println("User not found in allowlist")
            return
        }
        panic(err)
    }
    
    // Benutzerstatus prüfen
    if !user.IsActive() {
        // Benutzerstatus ist nicht aktiv, Anmeldung ablehnen
        fmt.Printf("User status is %s, cannot login\n", user.Status)
        return
    }
    
    // Benutzer existiert und ist aktiv, Anmeldeablauf fortsetzen
    fmt.Printf("User found: %s, Status: %s, Role: %s, Scopes: %v\n",
        user.UserID, user.Status, user.Role, user.Scope)
    
    // Als Nächstes: Herald aufrufen, um einen Bestätigungscode zu senden
    // ...
}
```

### Beispiel für einen vollständigen Anmeldeablauf (optionales Integrationsszenario)

In optionalen Integrationsszenarien kann der vollständige Anmeldeablauf wie folgt aussehen:

1. **Benutzer gibt eine Kennung ein** → Der Authentifizierungsdienst nimmt sie entgegen
2. **Authentifizierungsdienst → Warden**: Benutzerinformationen abfragen
   ```go
   user, err := wardenClient.GetUserByIdentifier(ctx, phone, mail, "")
   ```
3. **Benutzerstatus prüfen**: `user.Status == "active"` prüfen
4. **Authentifizierungsdienst → OTP-Dienst**: Challenge erstellen und Bestätigungscode senden (optional)
5. **Benutzer übermittelt den Bestätigungscode** → Der Authentifizierungsdienst nimmt ihn entgegen (optional)
6. **Authentifizierungsdienst → OTP-Dienst**: Bestätigungscode verifizieren (optional)
7. **Authentifizierungsdienst**: Sitzung ausstellen und mit `user.Scope` und `user.Role` die Autorisierungs-Header setzen

**Hinweis**: Warden kann eigenständig verwendet werden; der obige Integrationsablauf ist optional.

## Verwandte Dokumentation

- [OpenAPI-Spezifikation](../../openapi.yaml) – Vollständige OpenAPI-3.1-Spezifikation
- [Konfigurationsdokumentation](CONFIGURATION.md) – Erfahren Sie, wie Sie API-Key und weitere Optionen konfigurieren
- [Sicherheitsdokumentation](SECURITY.md) – Erfahren Sie mehr über Sicherheitsfunktionen und Best Practices
- [Architekturdokumentation](ARCHITECTURE.md) – Erfahren Sie mehr über die Architektur der Dienstintegration
