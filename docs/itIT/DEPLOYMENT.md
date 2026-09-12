# Documentazione di distribuzione

> 🌐 **Language / 语言**: [English](../enUS/DEPLOYMENT.md) | [中文](../zhCN/DEPLOYMENT.md) | [Français](../frFR/DEPLOYMENT.md) | [Italiano](DEPLOYMENT.md) | [日本語](../jaJP/DEPLOYMENT.md) | [Deutsch](../deDE/DEPLOYMENT.md) | [한국어](../koKR/DEPLOYMENT.md)

Questo documento spiega come distribuire il servizio Warden: distribuzione con Docker, distribuzione locale e altro ancora.

## Prerequisiti

- Go 1.27+ (vedere [go.mod](../../go.mod))
- Redis (per i lock distribuiti e la cache)
- Docker (facoltativo, per la distribuzione in container)

## Distribuzione con Docker

> 🚀 **Distribuzione rapida**: consulta la [directory degli esempi](../../example/README.md) / [示例目录](../../example/README.md) per configurazioni Docker Compose complete:
> - [Esempio semplice](../../example/basic/docker-compose.yml) / [简单示例](../../example/basic/docker-compose.yml) – Configurazione Docker Compose di base
> - [Esempio avanzato](../../example/advanced/docker-compose.yml) / [复杂示例](../../example/advanced/docker-compose.yml) – Configurazione completa con API simulata

### Usare l'immagine precompilata (consigliato)

Warden fornisce immagini Docker precompilate, scaricabili direttamente dal GitHub Container Registry (GHCR), senza bisogno di compilazione manuale:

```bash
# Scaricare l'immagine dell'ultima versione
docker pull ghcr.io/soulteary/warden:latest

# Avviare il container
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

> 💡 **Suggerimento**: usando immagini precompilate si parte subito, senza un ambiente di compilazione locale. Le immagini vengono aggiornate automaticamente, così si usa sempre l'ultima versione.

### Usare Docker Compose

1. **Preparare il file delle variabili d'ambiente**
   
   Se nella radice del progetto esiste un file `.env.example`, è possibile copiarlo:
   ```bash
   cp .env.example .env
   ```
   
   Se il file `.env.example` non esiste, si può creare manualmente un file `.env` con il contenuto seguente:
   ```env
   # Configurazione del server
   PORT=8081
   
   # Configurazione Redis
   REDIS=warden-redis:6379
   # Password Redis (facoltativa, si consigliano le variabili d'ambiente anziché il file di configurazione)
   # REDIS_PASSWORD=your-redis-password
   # Oppure usare un file per la password (più sicuro)
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # API dei dati remota
   CONFIG=http://example.com/api/data.json
   # Chiave di autenticazione dell'API di configurazione remota
   KEY=Bearer your-token-here
   
   # Configurazione delle attività
   INTERVAL=5
   
   # Modalità dell'applicazione
   MERGE_MODE=DEFAULT
   
   # Configurazione del client HTTP (facoltativa)
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # Chiave API (per l'autenticazione dell'API, obbligatoria in produzione)
   API_KEY=your-api-key-here
   
   # Lista di autorizzazione IP del controllo di integrità (facoltativa, separata da virgole)
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # Elenco degli IP dei proxy attendibili (facoltativo, separato da virgole, per ambienti con reverse proxy)
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # Livello di log (facoltativo)
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **Nota di sicurezza**: il file `.env` contiene informazioni sensibili. Non inserirlo nel controllo di versione. Il file `.env` è già ignorato da `.gitignore`. Usa il contenuto sopra come modello per creare il tuo file `.env`.

2. **Avviare il servizio**
```bash
docker-compose up -d
```

### Compilazione manuale dell'immagine

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### Avviare il container

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

## Distribuzione locale

### 1. Clonare il progetto

```bash
git clone <repository-url>
cd warden
```

### 2. Installare le dipendenze

```bash
go mod download
```

### 3. Configurare il file di dati locale

Creare un file `data.json` (vedere `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Nota**: il file `data.json` supporta i campi seguenti:
- `phone` (obbligatorio): numero di telefono dell'utente
- `mail` (obbligatorio): indirizzo e-mail dell'utente
- `user_id` (facoltativo): identificatore univoco dell'utente, generato automaticamente se non fornito
- `status` (facoltativo): stato dell'utente, ad esempio "active", "inactive", "suspended"; in assenza di valore il valore predefinito è "inactive"
- `scope` (facoltativo): array degli ambiti di autorizzazione dell'utente, ad esempio `["read", "write"]`
- `role` (facoltativo): ruolo dell'utente, ad esempio "admin", "user"

Per un esempio completo, consultare il file `data.example.json`.

### 4. Avviare il servizio

```bash
go run .
```

## Consigli per la distribuzione in produzione

### 1. Usare un reverse proxy

In produzione si consiglia di usare un reverse proxy come Nginx o Traefik:

**Esempio di configurazione Nginx**:
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

### 2. Usare HTTPS

Gli ambienti di produzione devono usare HTTPS. È possibile ottenerlo:

- Usando i certificati gratuiti di Let's Encrypt
- Usando un reverse proxy (come Nginx) per gestire SSL/TLS
- Configurando la variabile d'ambiente `TRUSTED_PROXY_IPS` per ottenere correttamente l'IP reale del client

### 3. Configurare il monitoraggio

- Usare Prometheus per raccogliere le metriche (tramite l'endpoint `/metrics`)
- Configurare i controlli di integrità (tramite l'endpoint `/health`)
- Predisporre la raccolta e l'analisi dei log

### 4. Distribuzione ad alta disponibilità

- Distribuire più istanze e usare un bilanciatore di carico per ripartire le richieste
- Usare un'istanza Redis condivisa per garantire la coerenza dei dati
- Configurare il riavvio automatico e il failover

### 5. Limiti di risorse

Configurare i limiti di risorse in Docker Compose o Kubernetes:

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

## Distribuzione con Kubernetes

### Distribuzione di base

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

## Ottimizzazione delle prestazioni

### 1. Configurazione Redis

- Usare la persistenza di Redis (RDB oppure AOF)
- Configurare limiti di memoria Redis adeguati
- Usare un cluster Redis (se necessario)

### 2. Configurazione dell'applicazione

- Regolare `HTTP_MAX_IDLE_CONNS` per ottimizzare il pool di connessioni
- Configurare un `INTERVAL` adeguato per bilanciare tempestività ed efficienza
- Usare una modalità di unione (`MERGE_MODE`) appropriata

### 3. Monitoraggio e messa a punto

In base ai risultati di un test di carico con wrk (test di 30 secondi, 16 thread, 100 connessioni):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

Regolare i parametri di configurazione in base al carico reale.

## Distribuzione con integrazione facoltativa (con Stargate/Herald)

Warden può essere distribuito e usato in modo autonomo, oppure integrato facoltativamente con Stargate e Herald. Di seguito alcuni esempi di configurazione per una distribuzione integrata facoltativa.

**Nota**: gli scenari di distribuzione integrata seguenti sono facoltativi; Warden può essere distribuito e usato in modo del tutto indipendente.

### Esempio di integrazione con Docker Compose

Configurazione completa di distribuzione integrata di Stargate + Warden + Herald:

```yaml
version: '3.8'

services:
  # Servizio Warden
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
      # Configurazione dell'autenticazione tra servizi (esempio con HMAC)
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

  # Redis di Warden
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

  # Servizio Stargate (esempio di configurazione)
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

  # Servizio Herald (esempio di configurazione)
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

  # Redis di Herald
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

### Configurazione delle variabili d'ambiente

Creare il file `.env`:

```bash
# Chiave API di Warden
WARDEN_API_KEY=your-warden-api-key-here

# Chiavi HMAC di Warden (formato JSON)
WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'

# Segreto HMAC usato da Stargate (corrisponde alla chiave in WARDEN_HMAC_KEYS)
WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef
```

### Configurazione di rete

Tutti i servizi devono trovarsi sulla stessa rete Docker per poter comunicare tra loro:

- **Warden**: resta in ascolto sulla porta `8081`, chiamato da Stargate
- **Stargate**: resta in ascolto sulla porta `8080`, funge da servizio forwardAuth per Traefik
- **Herald**: resta in ascolto sulla porta `8082`, chiamato da Stargate

### Dipendenze tra servizi

- **Stargate** dipende da **Warden** e **Herald**
- **Warden** dipende da **warden-redis** (facoltativo, se Redis è abilitato)
- **Herald** dipende da **herald-redis**

### Controlli di integrità

Tutti i servizi devono configurare controlli di integrità per garantirne il corretto funzionamento:

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### Consigli per l'ambiente di produzione

1. **Usare istanze Redis indipendenti**: Warden e Herald devono usare istanze Redis distinte per evitare conflitti di dati
2. **Configurare l'autenticazione tra servizi**: in produzione è obbligatorio configurare mTLS oppure la firma HMAC
3. **Usare servizi di gestione delle chiavi**: usare HashiCorp Vault o servizi analoghi per gestire chiavi e certificati
4. **Isolamento di rete**: usare le policy di rete di Docker per limitare gli accessi tra servizi
5. **Monitoraggio e log**: predisporre sistemi unificati di monitoraggio e raccolta dei log

### Distribuzione integrata su Kubernetes

Per la distribuzione su Kubernetes si consiglia di:

1. **Usare i Service**: creare un Service Kubernetes per ciascun servizio
2. **Usare ConfigMap e Secret**: conservarvi configurazione e chiavi
3. **Usare NetworkPolicy**: limitare gli accessi di rete tra servizi
4. **Usare Ingress**: configurare l'Ingress Traefik per instradare verso Stargate

Esempio di configurazione Kubernetes:

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

## Documentazione correlata

- [Documentazione di configurazione](CONFIGURATION.md) – Scopri le opzioni di configurazione dettagliate
- [Documentazione di sicurezza](SECURITY.md) – Scopri la configurazione di sicurezza e le buone pratiche
- [Documento di progettazione dell'architettura](ARCHITECTURE.md) – Comprendere l'architettura del sistema
- [Documentazione dell'API](API.md) – Scopri le interfacce dell'API e gli esempi di integrazione
