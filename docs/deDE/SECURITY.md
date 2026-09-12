# Sicherheitsdokumentation

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](SECURITY.md) | [한국어](../koKR/SECURITY.md)

Dieses Dokument erläutert die Sicherheitsfunktionen, die Sicherheitskonfiguration und die Best Practices von Warden.

## Umgesetzte Sicherheitsfunktionen

1. **API-Authentifizierung**: Unterstützt API-Key-Authentifizierung zum Schutz sensibler Endpunkte
2. **SSRF-Schutz**: Prüft URLs der Remote-Konfiguration streng, um Server-Side Request Forgery zu verhindern
3. **Eingabevalidierung**: Prüft alle Eingabeparameter streng, um Injection-Angriffe zu verhindern
4. **Ratenbegrenzung**: IP-basierte Ratenbegrenzung zum Schutz vor DDoS-Angriffen
5. **TLS-Prüfung**: Produktionsumgebungen erzwingen die Prüfung von TLS-Zertifikaten
6. **Fehlerbehandlung**: Produktionsumgebungen verbergen detaillierte Fehlerinformationen, um Informationsabfluss zu verhindern
7. **Sicherheits-Antwortheader**: Fügt automatisch sicherheitsrelevante HTTP-Antwortheader hinzu
8. **IP-Allow-Liste**: Unterstützt eine IP-Allow-Liste für die Health-Check-Endpunkte
9. **Validierung der Konfigurationsdatei**: Verhindert Path-Traversal-Angriffe
10. **Größenbegrenzung für JSON**: Begrenzt die Größe von JSON-Antwortkörpern, um Speichererschöpfungsangriffe zu verhindern
11. **Längenbegrenzung für Benutzerabfrageparameter**: Ein einzelner Parameter (`phone`/`mail`/`user_id`) darf 512 Byte nicht überschreiten, um DoS sowie das Aufblähen von Logs und Cache zu verhindern
12. **PII-Bereinigung im Audit-Log**: Die ins Audit geschriebene Kennung wird für phone/mail maskiert, damit bei einer Kompromittierung des Audit-Speichers keine personenbezogenen Daten preisgegeben werden

## Best Practices für Sicherheit

### 1. Konfiguration der Produktionsumgebung

**Erforderliche Konfiguration**:
- `ENVIRONMENT=production` **muss** gesetzt sein, um die Produktionshärtung zu aktivieren.
- Mindestens ein Verfahren zur Dienstauthentifizierung **muss** konfiguriert sein: `API_KEY`, HMAC v2 oder mTLS.
- `TRUSTED_PROXY_IPS` **muss** konfiguriert sein, damit die Client-IP korrekt ermittelt wird
- `HEALTH_CHECK_IP_WHITELIST` **muss** den Zugriff auf den Health-Check einschränken (oder `/health` und `/healthcheck` werden auf Netzwerk- bzw. Reverse-Proxy-Ebene beschränkt)
- `/metrics` **muss** eingeschränkt werden. Mit `ENVIRONMENT=production` ist die Authentifizierung standardmäßig erforderlich; belassen Sie es dabei (oder beschränken Sie den Pfad auf Reverse-Proxy-/Netzwerkebene) und setzen Sie in der Produktion **nicht** `WARDEN_METRICS_REQUIRE_AUTH=false`.

**Konfigurationsbeispiel**:
```bash
export API_KEY="your-strong-api-key-here"
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. Umgang mit vertraulichen Informationen

**Empfohlene Vorgehensweisen**:
- ✅ Passwörter und Schlüssel in Umgebungsvariablen ablegen
- ✅ Passwortdateien (`REDIS_PASSWORD_FILE`) für Redis-Passwörter verwenden
- ✅ In Konfigurationsdateien Platzhalter oder Kommentare verwenden
- ✅ Sicherstellen, dass die Dateirechte der Konfigurationsdateien korrekt gesetzt sind (etwa `chmod 600`)

**Nicht empfohlen**:
- ❌ Passwörter fest in Konfigurationsdateien hinterlegen
- ❌ Passwörter über Kommandozeilenargumente übergeben (sie erscheinen in der Prozessliste)
- ❌ Konfigurationsdateien mit vertraulichen Informationen in die Versionsverwaltung einchecken

**Beispiel**:
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # Umgebungsvariable REDIS_PASSWORD oder REDIS_PASSWORD_FILE verwenden

app:
  # api_key: ""  # Umgebungsvariable API_KEY verwenden
```

### 3. Netzwerksicherheit

**Erforderliche Konfiguration**:
- Produktionsumgebungen müssen HTTPS verwenden
- Firewallregeln konfigurieren, um den Zugriff einzuschränken
- Abhängigkeiten regelmäßig aktualisieren, um bekannte Schwachstellen zu beheben

**Empfohlene Konfiguration**:
- Einen Reverse Proxy (etwa Nginx) für SSL/TLS verwenden
- `TRUSTED_PROXY_IPS` konfigurieren, damit die echte Client-IP korrekt ermittelt wird
- Starke Passwörter und API-Keys verwenden
- `HTTP_INSECURE_TLS` deaktivieren (in der Produktion muss der Wert `false` sein)

### 4. Monitoring und Auditierung

**Empfohlene Vorgehensweisen**:
- Logs zu Sicherheitsereignissen überwachen
- Zugriffslogs regelmäßig prüfen
- Sicherheitsscanner in CI/CD einsetzen
- Alarmierungsmechanismen einrichten

**Verwaltung des Log-Levels**:
- In Produktionsumgebungen wird das Level `info` oder `warn` empfohlen
- Alle Änderungen des Log-Levels werden in den Sicherheits-Audit-Logs festgehalten
- Log-Level lassen sich dynamisch über die API `/log/level` anpassen (erfordert API-Key-Authentifizierung)

## API-Sicherheit

### API-Key-Authentifizierung

Einige API-Endpunkte erfordern eine Authentifizierung per API-Key:

**Endpunkte, die eine Authentifizierung erfordern**:
- `GET /` – Benutzerliste abrufen
- `GET /user` – Einzelnen Benutzer abfragen
- `GET /log/level` – Log-Level abfragen
- `POST /log/level` – Log-Level setzen

**Endpunkte ohne Authentifizierung** (müssen in der Produktion anderweitig geschützt werden):
- `GET /health` – Health-Check (`HEALTH_CHECK_IP_WHITELIST` oder Netzwerkisolation **muss** konfiguriert sein)
- `GET /healthcheck` – Health-Check (wie oben)
- `GET /metrics` – Prometheus-Metriken (für das Scraping **muss** ein API-Key gesetzt oder der Zugriff per Reverse Proxy/Netzwerk beschränkt werden; nicht öffentlich zugänglich machen)

**Authentifizierungsverfahren**:
1. **X-API-Key-Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization-Bearer-Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Ratenbegrenzung

Standardmäßig sind API-Anfragen durch eine Ratenbegrenzung geschützt:

- **Limit**: 60 Anfragen pro Minute
- **Zeitfenster**: 1 Minute
- **Bei Überschreitung**: Liefert `429 Too Many Requests`

Lässt sich über die Konfigurationsdatei anpassen:

```yaml
rate_limit:
  rate: 60  # Anfragen pro Minute
  window: 1m
```

### IP-Allow-Liste

Zwei Arten von IP-Allow-Listen werden unterstützt:

1. **Globale IP-Allow-Liste** (`IP_WHITELIST`):
   - Schränkt den Zugriff auf alle Endpunkte ein
   - Unterstützt das CIDR-Bereichsformat

2. **IP-Allow-Liste für den Health-Check** (`HEALTH_CHECK_IP_WHITELIST`):
   - Schränkt nur die Endpunkte `/health` und `/healthcheck` ein
   - Unterstützt das CIDR-Bereichsformat

**Konfigurationsbeispiel**:
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Datensicherheit

### Sicherheit der Remote-Konfigurations-API

- Remote-Konfigurations-APIs sollten Authentifizierungsmechanismen verwenden (Authorization-Header)
- Das HTTPS-Protokoll wird empfohlen
- TLS-Zertifikate der Remote-API prüfen (in der Produktion erforderlich)

### Redis-Sicherheit

- Redis sollte mit Passwortschutz konfiguriert sein
- Die Umgebungsvariablen `REDIS_PASSWORD` oder `REDIS_PASSWORD_FILE` verwenden
- Den Netzwerkzugriff auf Redis einschränken (nur vom Anwendungsserver zulassen)
- Redis regelmäßig aktualisieren, um bekannte Schwachstellen zu beheben

### Sicherheit der Datendatei

- Sicherstellen, dass die Dateirechte von `data.json` korrekt gesetzt sind
- Keine vertraulichen Daten in die Versionsverwaltung einchecken
- Datendateien regelmäßig sichern

## Sicherheits-Antwortheader

Warden fügt automatisch die folgenden sicherheitsrelevanten HTTP-Antwortheader hinzu:

- `X-Content-Type-Options: nosniff` – Verhindert MIME-Type-Sniffing
- `X-Frame-Options: DENY` – Verhindert Clickjacking
- `X-XSS-Protection: 1; mode=block` – XSS-Schutz

## Fehlerbehandlung

### Produktionsmodus

Im Produktionsmodus (`ENVIRONMENT=production`):

- Detaillierte Fehlerinformationen werden verborgen, um Informationsabfluss zu verhindern
- Es werden allgemeine Fehlermeldungen zurückgegeben
- Detaillierte Fehlerinformationen werden ausschließlich in den Logs festgehalten

### Entwicklungsmodus

Im Entwicklungsmodus:

- Detaillierte Fehlerinformationen werden zur Fehlersuche angezeigt
- Informationen zum Stacktrace sind enthalten

## Sicherheitsaudit

Hinweise zur Härtung und Überprüfung von Releases finden Sie unter [Release Security](../RELEASE_SECURITY.md).

## Meldung von Schwachstellen

Wenn Sie eine Sicherheitslücke entdecken, melden Sie diese bitte auf einem der folgenden Wege:

1. Legen Sie ein privates Security-Issue an (sofern unterstützt)
2. Senden Sie eine E-Mail an die Projektbetreuer
3. Legen Sie die Schwachstelle nicht öffentlich offen, bevor sie behoben ist

## Authentifizierung zwischen Diensten (optional)

Wenn Sie eine Integration mit anderen Diensten (etwa Stargate) wählen, kann die Authentifizierung zwischen Diensten die Sicherheit gewährleisten. **mTLS und HMAC sind umgesetzt**; die Reihenfolge der Authentifizierung ist **mTLS > HMAC > API-Key**. Warden unterstützt die folgenden Verfahren:

**Hinweis**: Wird Warden eigenständig betrieben, ist die Authentifizierung zwischen Diensten optional.

### mTLS (empfohlen)

Gegenseitige TLS-Zertifikate für die Authentifizierung verwenden; das bietet ein höheres Sicherheitsniveau.

**Konfiguration**:

1. **Zertifikate erzeugen**:
   ```bash
   # CA-Zertifikat erzeugen
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # Serverzertifikat für Warden erzeugen
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # Clientzertifikat für Stargate erzeugen
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Konfiguration von Warden** (Umgebungsvariablen):
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Konfiguration von Stargate**:
   - Pfad zum Clientzertifikat konfigurieren
   - Pfad zum CA-Zertifikat konfigurieren, um das Serverzertifikat von Warden zu prüfen

### HMAC-Signatur

Anfragen mit einer HMAC-SHA256-Signatur verifizieren; das ist einfacher zu betreiben.

**Signaturalgorithmus**:
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**Anfrageheader**:
- `X-Signature`: Wert der HMAC-Signatur
- `X-Timestamp`: Unix-Zeitstempel (Sekunden)
- `X-Key-Id`: Key-ID (Teil der Signatur, für eine sichere Schlüsselrotation)
- `X-Nonce`: Eindeutige 128-Bit-Nonce in Hexadezimalform
- `X-Signature-Version`: `v2`

**Konfiguration von Warden** (Umgebungsvariablen):
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # Zeitstempeltoleranz (Sekunden), Standard 60
export WARDEN_HMAC_ALLOW_V1=false          # Standard; nur während einer zeitlich begrenzten Legacy-Migration auf true setzen
```

Die Produktionskonfiguration verlangt, dass jedes HMAC-Geheimnis mindestens 32 Rohbytes umfasst.
Erzeugen Sie Geheimnisse mit einer kryptografisch sicheren Zufallsquelle und legen Sie sie in
einem Secret-Manager ab; verwenden Sie den obigen Beispielwert nicht erneut.

**Beispiel mit dem Go-SDK**:
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**Prüfregeln**:
- Warden prüft, ob der Zeitstempel innerhalb der Toleranz liegt (standardmäßig ±60 Sekunden)
- Warden prüft alle kanonischen Felder einschließlich der Key-ID und weist wiederverwendete Nonces zurück
- Schlägt die Signaturprüfung fehl, wird `401 Unauthorized` zurückgegeben

### Konfigurationspriorität

1. **mTLS**: Sind TLS-Zertifikate konfiguriert, wird zuerst mTLS verwendet
2. **HMAC**: Ist mTLS nicht konfiguriert, wird die HMAC-Signatur verwendet
3. **API-Key**: Ist keines von beiden konfiguriert, wird auf die API-Key-Authentifizierung zurückgegriffen (für Aufrufe zwischen Diensten nicht empfohlen)

### Sicherheitsempfehlungen

1. **Produktionsumgebung**: mTLS für die Authentifizierung zwischen Diensten wird dringend empfohlen
2. **Schlüsselverwaltung**: Dienste zur Schlüsselverwaltung (etwa HashiCorp Vault) für Schlüssel und Zertifikate nutzen
3. **Schlüsselrotation**: HMAC-Schlüssel und TLS-Zertifikate regelmäßig rotieren
4. **Netzwerkisolation**: Wo möglich, den Zugriff auf Warden per Netzwerkrichtlinie auf Stargate beschränken

## Verwandte Dokumentation

- [Konfigurationsdokumentation](CONFIGURATION.md) – Erfahren Sie mehr über sicherheitsrelevante Konfigurationsoptionen
- [Bereitstellungsdokumentation](DEPLOYMENT.md) – Erfahren Sie mehr über Empfehlungen für die Bereitstellung in der Produktion
- [API-Dokumentation](API.md) – Erfahren Sie mehr über die Sicherheitsfunktionen der API
- [Architekturdokumentation](ARCHITECTURE.md) – Erfahren Sie mehr über die Architektur der Dienstintegration
