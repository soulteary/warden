# API-Dokumentation

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](../frFR/API.md) | [Italiano](../itIT/API.md) | [日本語](../jaJP/API.md) | [Deutsch](API.md) | [한국어](../koKR/API.md)

Dieses Dokument enthält detaillierte Informationen zu allen von Warden bereitgestellten API-Endpunkten.

## OpenAPI-Dokumentation

Das Projekt bietet eine vollständige OpenAPI 3.0-Spezifikationsdokumentation in der Datei `openapi.yaml`.

Sie können die folgenden Tools verwenden, um die API anzuzeigen und zu testen:

1. **Swagger UI**: Öffnen Sie die Datei `openapi.yaml` mit [Swagger Editor](https://editor.swagger.io/)
2. **Postman**: Importieren Sie die Datei `openapi.yaml` in Postman
3. **Redoc**: Verwenden Sie Redoc, um eine schöne API-Dokumentationsseite zu generieren

## Authentifizierung

Einige API-Endpunkte erfordern eine API-Key-Authentifizierung. Sie können Authentifizierungsinformationen auf zwei Arten bereitstellen:

1. **X-API-Key-Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

Der API-Key kann über die Umgebungsvariable `API_KEY` oder das Kommandozeilenargument `--api-key` konfiguriert werden.

## API-Endpunkte

### Benutzerliste Abrufen

Alle Benutzer oder paginierte Benutzerliste abrufen.

**Anfrage**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**Abfrageparameter**:
- `page` (optional): Seitennummer, beginnend bei 1, Standardwert 1
- `page_size` (optional): Anzahl der Elemente pro Seite, Standardwert alle Daten (keine Paginierung)

**Hinweis**: Dieser Endpunkt erfordert eine API-Key-Authentifizierung.

**Antwort (keine Paginierung)**
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

**Antwort (mit Paginierung)**
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

### Einzelnen Benutzer Abrufen

Einen einzelnen Benutzer anhand der Telefonnummer, E-Mail-Adresse oder Benutzer-ID abfragen.

**Anfrage**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**Abfrageparameter** (genau einer muss angegeben werden):
- `phone`: Benutzertelefonnummer
- `mail`: Benutzer-E-Mail-Adresse
- `user_id`: Eindeutige Benutzerkennung

**Hinweis**: 
- Dieser Endpunkt erfordert eine API-Key-Authentifizierung
- Nur ein Abfrageparameter (`phone`, `mail` oder `user_id`) ist erlaubt

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
- `phone`: Benutzertelefonnummer
- `mail`: Benutzer-E-Mail-Adresse
- `user_id`: Eindeutige Benutzerkennung (automatisch generiert, wenn nicht angegeben)
- `status`: Benutzerstatus, mögliche Werte:
  - `"active"`: Aktiver Status, Benutzer kann sich anmelden und auf das System zugreifen
  - `"inactive"`: Inaktiver Status, Benutzer kann sich nicht anmelden
  - `"suspended"`: Gesperrter Status, Benutzer kann sich nicht anmelden
  - Standardwert `"inactive"`, wenn nicht gesetzt; `"active"` muss ausdrücklich angegeben werden, um die Anmeldung zu erlauben
- `scope`: Array des Benutzerberechtigungsbereichs (optional), verwendet für feingranulare Autorisierung, z.B. `["read", "write", "admin"]`
- `role`: Benutzerrolle (optional), z.B. `"admin"`, `"user"`, `"guest"`

**Hinweise**:
- Nur Benutzer mit `status` `"active"` können Authentifizierungsprüfungen bestehen
- Die Felder `scope` und `role` werden von Stargate verwendet, um Autorisierungsheader (`X-Auth-Scopes` und `X-Auth-Role`) für nachgelagerte Dienste zu setzen

**Antwort (Benutzer nicht gefunden)**
- **Statuscode**: `404 Not Found`
- **Antworttext**: `User not found`

**Fehlerantwort (fehlender Parameter)**
- **Statuscode**: `400 Bad Request`
- **Antworttext**: `Bad Request: missing identifier (phone, mail, or user_id)`

**Fehlerantwort (mehrere Parameter)**
- **Statuscode**: `400 Bad Request`
- **Antworttext**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### Gesundheitsprüfung

Dienststatus prüfen, einschließlich Redis-Verbindungsstatus, Datenladestatus usw.

**Anfrage**
```http
GET /health
GET /healthcheck
```

**Hinweis**: Dieser Endpunkt erfordert keine Authentifizierung, aber Zugriffs-IPs können über die Umgebungsvariable `HEALTH_CHECK_IP_WHITELIST` eingeschränkt werden.

**Antwort**
```json
{
    "status": "ok",
    "details": {
        "redis": "ok",
        "data_loaded": true,
        "user_count": 100
    },
    "mode": "DEFAULT"
}
```

**Statuscode**: `200 OK`

**Antwortfeldbeschreibungen**:
- `status`: Dienststatus, `"ok"` zeigt Normalzustand an
- `details.redis`: Redis-Verbindungsstatus, mögliche Werte:
  - `"ok"`: Redis ist normal
  - `"unavailable"`: Redis-Verbindung fehlgeschlagen (Fallback-Modus) oder Redis-Client ist nil
  - `"disabled"`: Redis ist explizit deaktiviert
- `details.data_loaded`: Ob Daten geladen wurden
- `details.user_count`: Aktuelle Benutzeranzahl
- `mode`: Aktueller Ausführungsmodus

### Protokollierungsstufen-Verwaltung

Protokollierungsstufen dynamisch abrufen und setzen.

#### Aktuelle Protokollierungsstufe Abrufen

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

**Hinweis**: Dieser Endpunkt erfordert eine API-Key-Authentifizierung.

#### Protokollierungsstufe Setzen

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

**Unterstützte Protokollierungsstufen**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**Antwort**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**Hinweis**: 
- Dieser Endpunkt erfordert eine API-Key-Authentifizierung
- Alle Protokollierungsstufen-Änderungsoperationen werden in Sicherheitsprüfprotokollen aufgezeichnet

### Prometheus-Metriken

Prometheus-Format-Überwachungsmetrikdaten abrufen.

**Anfrage**
```http
GET /metrics
```

**Antwort**: Prometheus-Format-Metrikdaten

**Hinweis**: Dieser Endpunkt erfordert keine Authentifizierung.

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

Alle API-Endpunkte können die folgenden Fehlerantworten zurückgeben:

### 401 Unauthorized

Wird zurückgegeben, wenn die API-Key-Authentifizierung fehlschlägt:

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

Wird zurückgegeben, wenn Anfragen das Rate-Limit überschreiten:

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

Im Produktionsmodus werden detaillierte Fehlerinformationen ausgeblendet, um Informationslecks zu verhindern.

## Rate Limiting

Standardmäßig sind API-Anfragen durch Rate Limiting geschützt:

- **Limit**: 60 Anfragen pro Minute
- **Fenster**: 1 Minute
- **Überschreitung**: Gibt `429 Too Many Requests` zurück

Rate Limiting kann über die Konfigurationsdatei angepasst werden:

```yaml
rate_limit:
  rate: 60  # Anfragen pro Minute
  window: 1m
```

## IP-Whitelist

IP-Whitelists können über die folgenden Umgebungsvariablen konfiguriert werden:

- `IP_WHITELIST`: Globale IP-Whitelist (schränkt den Zugriff auf alle Endpunkte ein)
- `HEALTH_CHECK_IP_WHITELIST`: Health-Check-Endpunkt-IP-Whitelist (schränkt nur `/health` und `/healthcheck` ein)

Unterstützt CIDR-Bereichsformat, mehrere IPs oder Bereiche durch Kommas getrennt:

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Antwortkomprimierung

Alle API-Antworten unterstützen automatische Komprimierung (gzip). Clients können die Komprimierung über den `Accept-Encoding: gzip`-Anfrageheader aktivieren.

## Verwandte Dokumentation

- [OpenAPI-Spezifikation](../openapi.yaml) - Vollständige OpenAPI 3.0-Spezifikation
- [Konfigurationsdokumentation](CONFIGURATION.md) - Erfahren Sie, wie Sie API-Key und andere Optionen konfigurieren
- [Sicherheitsdokumentation](SECURITY.md) - Erfahren Sie mehr über Sicherheitsfunktionen und Best Practices
