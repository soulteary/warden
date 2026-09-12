# Konfiguration

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

Dieses Dokument beschreibt die Konfigurationsoptionen von Warden im Detail, einschließlich Betriebsmodi, Formaten der Konfigurationsdateien, Umgebungsvariablen und mehr.

**Konfigurationspriorität**: Kommandozeilenargumente > Umgebungsvariablen > Konfigurationsdatei (YAML) > Standardwerte.

Eine **vollständige Optionstabelle** (YAML-Pfade, Umgebungsvariablen, Standardwerte, Validierungsregeln) finden Sie in der [zhCN-CONFIGURATION](../zhCN/CONFIGURATION.md). Zusammenfassung:

| Kategorie | YAML / Env | Hinweise |
|----------|------------|--------|
| Server | `server.*` / `PORT` | port, read_timeout, write_timeout, shutdown_timeout, idle_timeout, max_header_bytes |
| Redis | `redis.*` / `REDIS`, `REDIS_PASSWORD`, `REDIS_PASSWORD_FILE`, `REDIS_ENABLED` | addr, password, password_file, db; Redis standardmäßig aktiviert (`true`), außer bei ONLY_LOCAL ohne REDIS |
| Cache | `cache.ttl`, `cache.update_interval` | keine Überschreibung per Umgebungsvariable; update_interval standardmäßig 5s |
| Ratenbegrenzung | `rate_limit.rate`, `rate_limit.window` | standardmäßig 60/min, Zeitfenster 1m |
| HTTP-Client | `http.*` / `HTTP_TIMEOUT`, `HTTP_MAX_IDLE_CONNS`, `HTTP_INSECURE_TLS` | timeout, max_idle_conns, insecure_tls, max_retries, retry_delay |
| Remote | `remote.*` / `CONFIG`, `KEY`, `MERGE_MODE`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_RSA_PRIVATE_KEY_FILE`, `REMOTE_RSA_PRIVATE_KEY` | url, key, mode, decrypt_enabled, rsa_private_key_file |
| Task | `task.interval` | keine Überschreibung per Umgebungsvariable bei Verwendung einer Konfigurationsdatei; `INTERVAL` nur ohne Konfigurationsdatei verwenden |
| App | `app.*` / `API_KEY`, `DATA_FILE`, `DATA_DIR`, `RESPONSE_FIELDS` | mode, api_key, data_file, data_dir, response_fields |
| Tracing | `tracing.enabled`, `tracing.endpoint` / `OTLP_ENABLED`, `OTLP_ENDPOINT` | Bei `--config-file` wird `tracing` nicht aus dieser Datei gelesen, sofern `CONFIG_FILE` nicht auf denselben Pfad gesetzt ist |
| Dienstauthentifizierung | — / `WARDEN_HMAC_KEYS`, `WARDEN_HMAC_TIMESTAMP_TOLERANCE`, `WARDEN_TLS_*` | **Nur Umgebungsvariablen** (keine YAML-Schlüssel) |
| Health | — / `SNAPSHOT_MAX_AGE` | Maximal akzeptiertes Snapshot-Alter; Go-Duration, Standard `max(30s, 3 × Task-Intervall)` |

## Betriebsmodus (MERGE_MODE)

Das System unterstützt 7 Modi zum Zusammenführen von Daten, ausgewählt über `MERGE_MODE` (`MODE` ist veraltet):

| Modus | Beschreibung | Anwendungsfall |
|------|-------------|----------|
| `DEFAULT` | Historisches, remote-zuerst arbeitendes und tolerantes Verhalten | Abwärtskompatibilität für Bereitstellungen, die nie einen Modus ausgewählt haben |
| `REMOTE_FIRST` | Remote gewinnt, wenn das Laden erfolgreich ist; ein Remote-Fehler ist fatal und der zuletzt bekannte gute Snapshot bleibt erhalten | Strikte Bereitstellungen, in denen Remote maßgeblich ist |
| `ONLY_REMOTE` | Nur die entfernte Datenquelle verwenden | Vollständige Abhängigkeit von der Remote-Konfiguration |
| `ONLY_LOCAL` | Nur die lokale Konfigurationsdatei verwenden, **Redis standardmäßig deaktiviert** (wird aktiviert, wenn die Adresse `REDIS` ausdrücklich gesetzt ist oder `REDIS_ENABLED=true`) | Offline- oder Testumgebung |
| `LOCAL_FIRST` | Lokal zuerst; fehlen lokale Daten, werden sie durch Remote-Daten ergänzt | Lokale Konfiguration als primäre Quelle, Remote als Ergänzung |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Remote zuerst, Rückfall auf lokale Daten bei Remote-Fehlern erlaubt | Hochverfügbarkeitsszenarien |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Lokal zuerst, Rückfall auf Remote bei lokalen Fehlern erlaubt | Hybridmodus |

### Snapshot-Aktualität und Remote-Fehler

`REMOTE_FIRST` und `ONLY_REMOTE` sind strikte Modi. Schlägt eine geplante
Remote-Aktualisierung fehl, liefert Warden weiterhin den zuletzt bekannten guten
In-Memory-Snapshot aus, protokolliert den Aktualisierungsfehler und rückt das
`loaded_at` des Snapshots nicht vor. Der Health-Endpunkt liefert HTTP 503, sobald
der Snapshot `SNAPSHOT_MAX_AGE` überschreitet (Standard `max(30s, 3 × Task-Intervall)`).
Setzen Sie den Wert als Go-Duration, etwa `2m`.

`REMOTE_FIRST_ALLOW_REMOTE_FAILED` fällt ausdrücklich auf validierte lokale Daten
zurück, markiert den Snapshot als `degraded` und bleibt mit HTTP 200 bedienbar.
`LOCAL_FIRST` und `LOCAL_FIRST_ALLOW_REMOTE_FAILED` können gesund bleiben, wenn ihre
lokale Primärquelle erfolgreich ist, selbst wenn die Remote-Ergänzung nicht verfügbar
ist. `DEFAULT` behält aus Kompatibilitätsgründen das historische tolerante Verhalten
bei (einschließlich dieser Klartext-Semantik bei lokalem Erfolg); wählen Sie für neue
Produktionsbereitstellungen einen expliziten Modus. Fehler bei verschlüsselten
Remote-Quellen, die auf lokale Daten zurückfallen, werden in jedem toleranten Modus
als `degraded` gemeldet.

In Bereitstellungen mit mehreren Replikaten aktualisiert jedes Replikat seinen
prozesslokalen Cache und Snapshot. Ein verteilter Redis-Lock wählt lediglich den
Schreiber des gemeinsamen Redis-Caches, sodass auch Nicht-Schreiber ihre eigene
Snapshot-Aktualität vorantreiben.

### Konfigurationsmethoden

Sie können den Betriebsmodus auf folgende Arten festlegen:

**Kommandozeilenargumente**:
```bash
go run . --mode DEFAULT
```

**Umgebungsvariablen**:
```bash
export MERGE_MODE=DEFAULT
# MODE bleibt ein veralteter Kompatibilitätsalias.
```

**Konfigurationsdatei**:
```yaml
remote:
  mode: "DEFAULT"
# oder
app:
  mode: "DEFAULT"
```

## Format der Konfigurationsdateien

### Lokale Benutzerdatendatei (`data.json`)

Format der lokalen Benutzerdatendatei `data.json` (siehe `data.example.json`):

**Minimalformat** (nur Pflichtfelder):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Vollständiges Format** (mit allen optionalen Feldern):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    }
]
```

**Feldbeschreibungen**:
- `phone` (erforderlich): Telefonnummer des Benutzers
- `mail` (erforderlich): E-Mail-Adresse des Benutzers
- `user_id` (optional): Eindeutige Kennung des Benutzers; wird aus `phone` oder `mail` abgeleitet, wenn nicht angegeben
- `status` (optional): Benutzerstatus; fehlende Werte fallen sicher auf `"inactive"` zurück. Setzen Sie `"active"` ausdrücklich, um Zugriff zu erlauben.
- `scope` (optional): Array der Berechtigungsbereiche des Benutzers, standardmäßig ein leeres Array
- `role` (optional): Rolle des Benutzers, standardmäßig eine leere Zeichenkette

### Anwendungskonfigurationsdatei (`config.yaml`)

Konfigurationsdateien im YAML-Format werden unterstützt und über den Parameter `--config-file` angegeben:

```yaml
server:
  port: "8081"
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 5s
  max_header_bytes: 1048576  # 1 MB
  idle_timeout: 120s

redis:
  addr: "localhost:6379"
  password: ""  # Empfohlen wird die Umgebungsvariable REDIS_PASSWORD oder REDIS_PASSWORD_FILE
  password_file: ""  # Pfad zur Passwortdatei (höhere Priorität als password)
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60  # Anfragen pro Minute
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # Nur für die Entwicklung
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # Remote-Antwort per RSA entschlüsseln (zusammen mit rsa_private_key_file oder REMOTE_RSA_PRIVATE_KEY verwenden)
  rsa_private_key_file: ""    # Pfad zur PEM-Datei (oder Umgebungsvariable REMOTE_RSA_PRIVATE_KEY für inline-PEM)

task:
  interval: 5s

app:
  mode: "DEFAULT"  # Datenzusammenführungsmodus; die Produktionsrichtlinie wird über ENVIRONMENT gewählt
  api_key: ""      # Empfohlen wird die Umgebungsvariable API_KEY
  data_file: "./data.json"
  data_dir: ""     # Optional: alle *.json im Verzeichnis zusammenführen (kann mit data_file kombiniert werden)
  response_fields: []  # Optional: Whitelist für Antwortfelder der API; leer = alle Felder

tracing:
  enabled: false
  endpoint: ""     # z. B. "http://localhost:4318"
```

**Konfigurationspriorität**: Kommandozeilenargumente > Umgebungsvariablen > Konfigurationsdatei > Standardwerte.

**Hinweis zum Tracing**: Bei Verwendung von `--config-file` liest das Hauptprogramm den Abschnitt `tracing` nicht aus dieser Datei, sofern nicht die Umgebungsvariable `CONFIG_FILE` auf denselben Pfad gesetzt ist oder Sie `OTLP_ENABLED` + `OTLP_ENDPOINT` verwenden.

Siehe die Beispieldatei: [config.example.yaml](../../config.example.yaml).

## Kommandozeilenargumente

```bash
go run . \
  --port 8081 \                    # Port des Webdienstes (Standard: 8081)
  --redis localhost:6379 \         # Redis-Adresse (Standard: localhost:6379)
  --redis-password "password" \    # Redis-Passwort (optional, Umgebungsvariablen empfohlen)
  --redis-enabled=true \           # Redis aktivieren/deaktivieren (Standard: true)
  --config http://example.com/api \ # URL der Remote-Konfiguration
  --key "Bearer token" \           # Authentifizierungs-Header der Remote-Konfiguration
  --interval 5 \                   # Intervall der geplanten Aufgabe (Sekunden, Standard: 5)
  --mode DEFAULT \                 # Betriebsmodus (siehe Beschreibung oben)
  --http-timeout 5 \               # Timeout für HTTP-Anfragen (Sekunden, Standard: 5)
  --http-max-idle-conns 100 \     # Maximale Anzahl inaktiver HTTP-Verbindungen (Standard: 100)
  --http-insecure-tls \           # TLS-Zertifikatsprüfung überspringen (nur für die Entwicklung)
  --api-key "your-secret-api-key" \ # API-Key für die Authentifizierung (optional, Umgebungsvariablen empfohlen)
  --config-file config.yaml        # Pfad zur Konfigurationsdatei (unterstützt YAML-Format)
```

**Hinweise**:
- Unterstützung von Konfigurationsdateien: Mit dem Parameter `--config-file` können Sie eine Konfigurationsdatei im YAML-Format angeben
- Sicherheit des Redis-Passworts: Verwenden Sie besser die Umgebungsvariablen `REDIS_PASSWORD` oder `REDIS_PASSWORD_FILE` statt Kommandozeilenargumente
- TLS-Zertifikatsprüfung: `--http-insecure-tls` ist ausschließlich für Entwicklungsumgebungen gedacht und sollte in der Produktion nicht verwendet werden

## Umgebungsvariablen

Die Konfiguration über Umgebungsvariablen wird unterstützt und hat eine niedrigere Priorität als Kommandozeilenargumente. Die vollständige Optionstabelle (einschließlich Validierungsregeln) finden Sie in der [zhCN-CONFIGURATION](../zhCN/CONFIGURATION.md).

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis-Passwort (optional)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Pfad zur Redis-Passwortdatei (optional; Priorität: REDIS_PASSWORD > REDIS_PASSWORD_FILE > Konfiguration)
export REDIS_ENABLED=true               # Redis aktivieren/deaktivieren (optional, Standard: true, unterstützt true/false/1/0)
                                        # Hinweis: Im Modus ONLY_LOCAL ist der Standard false
                                        #       Ist jedoch eine REDIS-Adresse ausdrücklich gesetzt, wird Redis automatisch aktiviert
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MERGE_MODE=DEFAULT
export DATA_FILE=./data.json          # Pfad zur lokalen Benutzerdatendatei
export DATA_DIR=                      # Optional: Verzeichnis, in dem alle *.json zusammengeführt werden (kann mit DATA_FILE kombiniert werden)
export RESPONSE_FIELDS=               # Optional: Whitelist für Antwortfelder der API (kommagetrennt, z. B. phone,mail,user_id,status,name); leer = alle
export REMOTE_DECRYPT_ENABLED=false   # Optional: Remote-Antwort per RSA entschlüsseln
export REMOTE_RSA_PRIVATE_KEY_FILE=   # Optional: Pfad zur RSA-Privatschlüssel-PEM-Datei (oder REMOTE_RSA_PRIVATE_KEY für inline-PEM)
export REMOTE_RSA_PRIVATE_KEY=        # Optional: inline RSA-Privatschlüssel als PEM (wird verwendet, wenn REMOTE_RSA_PRIVATE_KEY_FILE nicht gesetzt ist)
export HTTP_TIMEOUT=5                  # Timeout für HTTP-Anfragen (Sekunden)
export HTTP_MAX_IDLE_CONNS=100         # Maximale Anzahl inaktiver HTTP-Verbindungen
export HTTP_INSECURE_TLS=false         # Ob die TLS-Zertifikatsprüfung übersprungen wird (true/false oder 1/0)
export API_KEY="your-secret-api-key"   # API-Key für die Authentifizierung (dringend empfohlen)
export CONFIG_FILE=config.yaml         # Optional; lädt Tracing aus YAML, wenn `--config-file` nicht verwendet wird, oder aktiviert Tracing aus derselben Datei wie `--config-file`
export OTLP_ENABLED=false              # OpenTelemetry aktivieren (true/false oder 1/0)
export OTLP_ENDPOINT=http://localhost:4318  # OTLP-Endpunkt (erforderlich, wenn OTLP_ENABLED true ist)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # Liste vertrauenswürdiger Proxy-IPs (kommagetrennt)
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # IP-Allow-Liste für den Health-Check-Endpunkt (optional)
export IP_WHITELIST="192.168.1.0/24"  # Globale IP-Allow-Liste (optional)
export LOG_LEVEL="info"                # Log-Level (optional, Standard: info, Optionen: trace, debug, info, warn, error, fatal, panic)
export WARDEN_HMAC_KEYS='{"key-id":"0123456789abcdef0123456789abcdef"}'  # Produktionsgeheimnisse benötigen mindestens 32 Byte
export WARDEN_HMAC_ALLOW_V1=false                # Standard: false; nur während einer zeitlich begrenzten v1-Migration auf true setzen
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # HMAC-Zeitstempeltoleranz (Sekunden)
export WARDEN_TLS_CERT=/path/to/warden.crt    # Dienstauthentifizierung: Server-TLS-Zertifikat (aktiviert zusammen mit KEY TLS)
export WARDEN_TLS_KEY=/path/to/warden.key     # Privater Server-TLS-Schlüssel
export WARDEN_TLS_CA=/path/to/ca.crt          # Client-CA (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # Clientzertifikat erforderlich (mTLS)
```

**Priorität der Umgebungsvariablen**:
- Redis-Passwort: `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > Kommandozeilenargument `--redis-password`

**Hinweise zur Sicherheitskonfiguration**:
- `API_KEY`: Schützt sensible Endpunkte (`/`, `/log/level`); für Produktionsumgebungen dringend empfohlen
- `TRUSTED_PROXY_IPS`: Konfigurieren Sie vertrauenswürdige Reverse-Proxy-IPs, damit die echte Client-IP korrekt ermittelt wird
- `HEALTH_CHECK_IP_WHITELIST`: Schränkt die Zugriffs-IPs des Health-Check-Endpunkts ein (optional, unterstützt CIDR-Bereiche)
- `IP_WHITELIST`: Globale IP-Allow-Liste (optional, unterstützt CIDR-Bereiche)

## Anforderungen an die Remote-Konfigurations-API

Die Remote-Konfigurations-API sollte ein JSON-Array im gleichen Format zurückgeben und optional die Authentifizierung per Authorization-Header unterstützen.

Das Antwortformat der API sollte dem Format der Datei `data.json` entsprechen:

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write"],
        "role": "admin"
    }
]
```

Ist die Umgebungsvariable `KEY` oder der Parameter `--key` konfiguriert, wird den Anfragen automatisch der Header `Authorization` hinzugefügt:

```http
Authorization: Bearer your-token-here
```

## Optionale Konfiguration der Dienstintegration

Wenn Sie eine Integration mit anderen Diensten (etwa Stargate) wählen, kann die Authentifizierung zwischen Diensten konfiguriert werden. Nachfolgend die relevanten Konfigurationspunkte:

**Hinweis**: Wird Warden eigenständig betrieben, sind die folgenden Konfigurationen optional.

### mTLS-Konfiguration (empfohlen)

Gegenseitige TLS-Zertifikate für die Authentifizierung zwischen Diensten verwenden. **Es werden nur Umgebungsvariablen unterstützt** (keine YAML-Schlüssel in der Anwendungskonfiguration):

```bash
# Serverzertifikat von Warden
export WARDEN_TLS_CERT=/path/to/warden.crt
export WARDEN_TLS_KEY=/path/to/warden.key
export WARDEN_TLS_CA=/path/to/ca.crt

# Clientzertifikat verlangen (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
```

### HMAC-Signaturkonfiguration

HMAC-SHA256-Signaturen für die Authentifizierung zwischen Diensten verwenden. **Es werden nur Umgebungsvariablen unterstützt** (keine YAML-Schlüssel):

```bash
# HMAC-Schlüssel (JSON-Format, unterstützt mehrere Schlüssel für die Rotation)
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef","key-id-2":"abcdef0123456789abcdef0123456789"}'

# Zeitstempeltoleranz (Sekunden), Standard 60, wenn HMAC-Schlüssel gesetzt sind
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60

# Das alte v1 ist standardmäßig deaktiviert. Nur während der Migration alter Aufrufer aktivieren.
export WARDEN_HMAC_ALLOW_V1=false
```

### Konfiguration der Aufrufe durch Stargate

Stargate muss die Adresse des Warden-Dienstes und die Authentifizierungsinformationen konfigurieren:

**Beispielkonfiguration für Stargate** (Umgebungsvariablen):
```bash
# Adresse des Warden-Dienstes
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# Authentifizierungsverfahren zwischen Diensten (mTLS oder HMAC)
export STARGATE_WARDEN_AUTH_TYPE=hmac

# HMAC-Konfiguration (bei Verwendung von HMAC)
export STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
export STARGATE_WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef

# mTLS-Konfiguration (bei Verwendung von mTLS)
export STARGATE_WARDEN_TLS_CERT=/path/to/stargate.crt
export STARGATE_WARDEN_TLS_KEY=/path/to/stargate.key
export STARGATE_WARDEN_TLS_CA=/path/to/ca.crt
```

### Konfigurationspriorität

1. **mTLS**: Sind TLS-Zertifikate konfiguriert, wird zuerst mTLS verwendet
2. **HMAC**: Ist mTLS nicht konfiguriert, wird die HMAC-Signatur verwendet
3. **API-Key**: Ist keines von beiden konfiguriert, wird auf die API-Key-Authentifizierung zurückgegriffen (für Aufrufe zwischen Diensten nicht empfohlen)

### Konfigurationsvalidierung

Beim Start prüft Warden die Konfiguration der Authentifizierung zwischen Diensten:

- Eine unvollständige TLS-Konfiguration wird abgelehnt: Zertifikat und Schlüssel müssen gemeinsam gesetzt sein; mTLS erfordert zusätzlich eine Client-CA
- Ist HMAC konfiguriert, wird das Schlüsselformat geprüft
- Unter `ENVIRONMENT=production` verweigert der Dienst den Start, sofern nicht API-Key, HMAC oder mTLS als Authentifizierung konfiguriert ist

## Ausführliche Konfigurationsdokumentation

Ausführlichere Informationen zu den Mechanismen der Parameterauflösung, den Prioritätsregeln und Anwendungsbeispielen finden Sie unter:

- [Entwurfsdokument zur Parameterauflösung](CONFIG_PARSING.md) – Ausführliche Dokumentation zum Mechanismus der Parameterauflösung
- [Architekturdokument](ARCHITECTURE.md) – Verstehen Sie die Gesamtarchitektur und die Auswirkungen der Konfiguration
- [Sicherheitsdokumentation](SECURITY.md) – Erfahren Sie mehr über die Details der Authentifizierung zwischen Diensten
