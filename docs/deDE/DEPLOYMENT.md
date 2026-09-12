# Bereitstellungsdokumentation

> 🌐 **Language / 语言**: [English](../enUS/DEPLOYMENT.md) | [中文](../zhCN/DEPLOYMENT.md) | [Français](../frFR/DEPLOYMENT.md) | [Italiano](../itIT/DEPLOYMENT.md) | [日本語](../jaJP/DEPLOYMENT.md) | [Deutsch](DEPLOYMENT.md) | [한국어](../koKR/DEPLOYMENT.md)

Dieses Dokument erklärt, wie der Warden-Dienst bereitgestellt wird – per Docker, lokal und mehr.

## Voraussetzungen

- Go 1.27+ (siehe [go.mod](../../go.mod))
- Redis (für verteilte Locks und Caching)
- Docker (optional, für die Bereitstellung im Container)

## Bereitstellung mit Docker

> 🚀 **Schnelle Bereitstellung**: Im [Beispielverzeichnis](../../example/README.md) / [示例目录](../../example/README.md) finden Sie vollständige Beispielkonfigurationen für Docker Compose:
> - [Einfaches Beispiel](../../example/basic/docker-compose.yml) / [简单示例](../../example/basic/docker-compose.yml) – Grundlegende Docker-Compose-Konfiguration
> - [Fortgeschrittenes Beispiel](../../example/advanced/docker-compose.yml) / [复杂示例](../../example/advanced/docker-compose.yml) – Vollständige Konfiguration inklusive Mock-API

### Vorgefertigtes Image verwenden (empfohlen)

Warden stellt vorgefertigte Docker-Images bereit, die sich direkt aus der GitHub Container Registry (GHCR) beziehen lassen – ein manueller Build ist nicht nötig:

```bash
# Image der neuesten Version beziehen
docker pull ghcr.io/soulteary/warden:latest

# Container starten
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api/data.json \
  -e KEY="Bearer your-token-here" \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Tipp**: Mit vorgefertigten Images können Sie ohne lokale Build-Umgebung sofort loslegen. Die Images werden automatisch aktualisiert, sodass Sie stets die neueste Version verwenden.

### Docker Compose verwenden

1. **Datei mit Umgebungsvariablen vorbereiten**
   
   Existiert im Projektstammverzeichnis eine Datei `.env.example`, können Sie diese kopieren:
   ```bash
   cp .env.example .env
   ```
   
   Fehlt die Datei `.env.example`, können Sie eine `.env`-Datei mit folgendem Inhalt anlegen:
   ```env
   # Serverkonfiguration
   PORT=8081
   
   # Redis-Konfiguration
   REDIS=warden-redis:6379
   # Redis-Passwort (optional, Umgebungsvariablen statt Konfigurationsdatei empfohlen)
   # REDIS_PASSWORD=your-redis-password
   # Oder Passwortdatei verwenden (sicherer)
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # Entfernte Daten-API
   CONFIG=http://example.com/api/data.json
   # Authentifizierungsschlüssel der Remote-Konfigurations-API
   KEY=Bearer your-token-here
   
   # Task-Konfiguration
   INTERVAL=5
   
   # Anwendungsmodus
   MERGE_MODE=DEFAULT
   
   # Konfiguration des HTTP-Clients (optional)
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # API-Key (für die API-Authentifizierung, in der Produktion erforderlich)
   API_KEY=your-api-key-here
   
   # IP-Allow-Liste für den Health-Check (optional, kommagetrennt)
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # Liste vertrauenswürdiger Proxy-IPs (optional, kommagetrennt, für Reverse-Proxy-Umgebungen)
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # Log-Level (optional)
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **Sicherheitshinweis**: Die Datei `.env` enthält vertrauliche Informationen. Committen Sie sie nicht in die Versionsverwaltung. Die Datei `.env` wird bereits von `.gitignore` ignoriert. Verwenden Sie den obigen Inhalt als Vorlage für Ihre `.env`-Datei.

2. **Dienst starten**
```bash
docker-compose up -d
```

### Image manuell bauen

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### Container starten

```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api \
  -e KEY="Bearer token" \
  warden-release
```

## Lokale Bereitstellung

### 1. Projekt klonen

```bash
git clone <repository-url>
cd warden
```

### 2. Abhängigkeiten installieren

```bash
go mod download
```

### 3. Lokale Datendatei konfigurieren

Legen Sie eine Datei `data.json` an (siehe `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Hinweis**: Die Datei `data.json` unterstützt die folgenden Felder:
- `phone` (erforderlich): Telefonnummer des Benutzers
- `mail` (erforderlich): E-Mail-Adresse des Benutzers
- `user_id` (optional): Eindeutige Kennung des Benutzers, wird automatisch erzeugt, wenn nicht angegeben
- `status` (optional): Benutzerstatus, etwa „active“, „inactive“, „suspended“; fehlende Werte gelten standardmäßig als „inactive“
- `scope` (optional): Array der Berechtigungsbereiche des Benutzers, etwa `["read", "write"]`
- `role` (optional): Rolle des Benutzers, etwa „admin“, „user“

Ein vollständiges Beispiel finden Sie in der Datei `data.example.json`.

### 4. Dienst starten

```bash
go run .
```

## Empfehlungen für die Bereitstellung in der Produktion

### 1. Reverse Proxy verwenden

In der Produktion empfiehlt sich ein Reverse Proxy wie Nginx oder Traefik:

**Beispielkonfiguration für Nginx**:
```nginx
upstream warden {
    server localhost:8081;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://warden;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. HTTPS verwenden

Produktionsumgebungen müssen HTTPS verwenden. Das lässt sich erreichen durch:

- Kostenlose Zertifikate von Let's Encrypt
- Einen Reverse Proxy (etwa Nginx), der SSL/TLS übernimmt
- Konfiguration der Umgebungsvariable `TRUSTED_PROXY_IPS`, damit die echte Client-IP korrekt ermittelt wird

### 3. Monitoring einrichten

- Prometheus zum Sammeln von Metriken verwenden (über den Endpunkt `/metrics`)
- Health-Checks konfigurieren (über den Endpunkt `/health`)
- Logsammlung und -auswertung einrichten

### 4. Hochverfügbare Bereitstellung

- Mehrere Instanzen bereitstellen und Anfragen per Load Balancer verteilen
- Eine gemeinsame Redis-Instanz verwenden, um Datenkonsistenz sicherzustellen
- Automatischen Neustart und Failover konfigurieren

### 5. Ressourcenbegrenzungen

Konfigurieren Sie Ressourcenbegrenzungen in Docker Compose oder Kubernetes:

```yaml
services:
  warden:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Bereitstellung mit Kubernetes

### Grundlegende Bereitstellung

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "redis-service:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: warden-service
spec:
  selector:
    app: warden
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  type: LoadBalancer
```

## Leistungsoptimierung

### 1. Redis-Konfiguration

- Redis-Persistenz verwenden (RDB oder AOF)
- Passende Speichergrenzen für Redis konfigurieren
- Redis-Cluster verwenden (falls erforderlich)

### 2. Anwendungskonfiguration

- `HTTP_MAX_IDLE_CONNS` anpassen, um den Verbindungspool zu optimieren
- Ein passendes `INTERVAL` konfigurieren, um Aktualität und Effizienz auszubalancieren
- Einen geeigneten Zusammenführungsmodus (`MERGE_MODE`) verwenden

### 3. Monitoring und Feinabstimmung

Basierend auf Lasttestergebnissen mit wrk (30 Sekunden, 16 Threads, 100 Verbindungen):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

Passen Sie die Konfigurationsparameter an die tatsächliche Last an.

## Optionale Bereitstellung mit Integration (mit Stargate/Herald)

Warden kann eigenständig bereitgestellt und betrieben oder optional mit Stargate und Herald integriert werden. Nachfolgend Beispielkonfigurationen für eine optionale Integrationsbereitstellung.

**Hinweis**: Die folgenden Integrationsszenarien sind optional; Warden lässt sich vollständig unabhängig bereitstellen und nutzen.

### Beispiel für die Integration mit Docker Compose

Vollständige Bereitstellungskonfiguration für die Integration von Stargate + Warden + Herald:

```yaml
version: '3.8'

services:
  # Warden-Dienst
  warden:
    image: ghcr.io/soulteary/warden:latest
    container_name: warden
    ports:
      - "8081:8081"
    networks:
      - auth-network
    environment:
      - PORT=8081
      - REDIS=warden-redis:6379
      - API_KEY=${WARDEN_API_KEY}
      - MERGE_MODE=DEFAULT
      # Konfiguration der Authentifizierung zwischen Diensten (HMAC-Beispiel)
      - WARDEN_HMAC_KEYS=${WARDEN_HMAC_KEYS}
      - WARDEN_HMAC_TIMESTAMP_TOLERANCE=60
    volumes:
      - ./warden-data.json:/app/data.json:ro
    healthcheck:
      test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
      interval: 10s
      timeout: 1s
      retries: 3
    depends_on:
      - warden-redis

  # Redis für Warden
  warden-redis:
    image: redis:7.4-alpine
    container_name: warden-redis
    networks:
      - auth-network
    volumes:
      - warden-redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 1s
      retries: 3

  # Stargate-Dienst (Beispielkonfiguration)
  stargate:
    image: ghcr.io/soulteary/stargate:latest
    container_name: stargate
    ports:
      - "8080:8080"
    networks:
      - auth-network
    environment:
      - STARGATE_WARDEN_BASE_URL=http://warden:8081
      - STARGATE_WARDEN_AUTH_TYPE=hmac
      - STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
      - STARGATE_WARDEN_HMAC_SECRET=${WARDEN_HMAC_SECRET}
      - STARGATE_HERALD_BASE_URL=http://herald:8082
    depends_on:
      - warden
      - herald

  # Herald-Dienst (Beispielkonfiguration)
  herald:
    image: ghcr.io/soulteary/herald:latest
    container_name: herald
    ports:
      - "8082:8082"
    networks:
      - auth-network
    environment:
      - HERALD_REDIS_URL=redis://herald-redis:6379
    depends_on:
      - herald-redis

  # Redis für Herald
  herald-redis:
    image: redis:7.4-alpine
    container_name: herald-redis
    networks:
      - auth-network
    volumes:
      - herald-redis-data:/data

networks:
  auth-network:
    driver: bridge

volumes:
  warden-redis-data:
  herald-redis-data:
```

### Konfiguration der Umgebungsvariablen

Legen Sie eine `.env`-Datei an:

```bash
# API-Key von Warden
WARDEN_API_KEY=your-warden-api-key-here

# HMAC-Schlüssel von Warden (JSON-Format)
WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'

# Von Stargate verwendetes HMAC-Geheimnis (entspricht dem Schlüssel in WARDEN_HMAC_KEYS)
WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef
```

### Netzwerkkonfiguration

Alle Dienste sollten sich im selben Docker-Netzwerk befinden, damit sie miteinander kommunizieren können:

- **Warden**: lauscht auf Port `8081`, wird von Stargate aufgerufen
- **Stargate**: lauscht auf Port `8080`, dient als forwardAuth-Dienst für Traefik
- **Herald**: lauscht auf Port `8082`, wird von Stargate aufgerufen

### Abhängigkeiten der Dienste

- **Stargate** hängt von **Warden** und **Herald** ab
- **Warden** hängt von **warden-redis** ab (optional, falls Redis aktiviert ist)
- **Herald** hängt von **herald-redis** ab

### Health-Checks

Alle Dienste sollten Health-Checks konfigurieren, um den ordnungsgemäßen Betrieb sicherzustellen:

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### Empfehlungen für die Produktionsumgebung

1. **Eigenständige Redis-Instanzen verwenden**: Warden und Herald sollten eigenständige Redis-Instanzen verwenden, um Datenkonflikte zu vermeiden
2. **Authentifizierung zwischen Diensten konfigurieren**: In der Produktion müssen mTLS oder HMAC-Signaturen konfiguriert sein
3. **Dienste zur Schlüsselverwaltung nutzen**: Verwenden Sie HashiCorp Vault oder vergleichbare Dienste zur Verwaltung von Schlüsseln und Zertifikaten
4. **Netzwerkisolation**: Beschränken Sie den Zugriff zwischen Diensten über Docker-Netzwerkrichtlinien
5. **Monitoring und Logging**: Richten Sie einheitliche Monitoring- und Logsammelsysteme ein

### Integrationsbereitstellung mit Kubernetes

Für die Bereitstellung in Kubernetes empfiehlt sich Folgendes:

1. **Services verwenden**: Legen Sie für jeden Dienst einen Kubernetes-Service an
2. **ConfigMap und Secret verwenden**: Konfiguration und Schlüssel dort ablegen
3. **NetworkPolicy verwenden**: Netzwerkzugriffe zwischen Diensten einschränken
4. **Ingress verwenden**: Traefik-Ingress so konfigurieren, dass er auf Stargate routet

Beispielkonfiguration für Kubernetes:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: warden
spec:
  selector:
    app: warden
  ports:
    - port: 8081
      targetPort: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: ghcr.io/soulteary/warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "warden-redis:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        - name: WARDEN_HMAC_KEYS
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: hmac-keys
```

## Verwandte Dokumentation

- [Konfigurationsdokumentation](CONFIGURATION.md) – Erfahren Sie mehr über die ausführlichen Konfigurationsoptionen
- [Sicherheitsdokumentation](SECURITY.md) – Erfahren Sie mehr über Sicherheitskonfiguration und Best Practices
- [Architekturdokument](ARCHITECTURE.md) – Verstehen Sie die Systemarchitektur
- [API-Dokumentation](API.md) – Erfahren Sie mehr über die API-Schnittstellen und Integrationsbeispiele
