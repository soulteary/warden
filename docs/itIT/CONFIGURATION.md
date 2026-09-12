# Configurazione

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

Questo documento descrive in dettaglio le opzioni di configurazione di Warden: modalità di esecuzione, formati dei file di configurazione, variabili d'ambiente e altro ancora.

**Priorità di configurazione**: argomenti da riga di comando > variabili d'ambiente > file di configurazione (YAML) > valori predefiniti.

Per una **tabella completa delle opzioni** (percorsi YAML, variabili d'ambiente, valori predefiniti, regole di convalida), consultare la [CONFIGURATION zhCN](../zhCN/CONFIGURATION.md). Riepilogo:

| Categoria | YAML / Env | Note |
|----------|------------|--------|
| Server | `server.*` / `PORT` | port, read_timeout, write_timeout, shutdown_timeout, idle_timeout, max_header_bytes |
| Redis | `redis.*` / `REDIS`, `REDIS_PASSWORD`, `REDIS_PASSWORD_FILE`, `REDIS_ENABLED` | addr, password, password_file, db; Redis abilitato per impostazione predefinita (`true`), tranne in ONLY_LOCAL senza REDIS |
| Cache | `cache.ttl`, `cache.update_interval` | nessuna sovrascrittura tramite variabili d'ambiente; update_interval predefinito 5s |
| Limitazione di frequenza | `rate_limit.rate`, `rate_limit.window` | predefinito 60/min, finestra di 1m |
| Client HTTP | `http.*` / `HTTP_TIMEOUT`, `HTTP_MAX_IDLE_CONNS`, `HTTP_INSECURE_TLS` | timeout, max_idle_conns, insecure_tls, max_retries, retry_delay |
| Remoto | `remote.*` / `CONFIG`, `KEY`, `MERGE_MODE`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_RSA_PRIVATE_KEY_FILE`, `REMOTE_RSA_PRIVATE_KEY` | url, key, mode, decrypt_enabled, rsa_private_key_file |
| Attività | `task.interval` | nessuna sovrascrittura tramite variabili d'ambiente quando si usa un file di configurazione; usare `INTERVAL` solo senza file di configurazione |
| Applicazione | `app.*` / `API_KEY`, `DATA_FILE`, `DATA_DIR`, `RESPONSE_FIELDS` | mode, api_key, data_file, data_dir, response_fields |
| Tracciamento | `tracing.enabled`, `tracing.endpoint` / `OTLP_ENABLED`, `OTLP_ENDPOINT` | Con `--config-file`, `tracing` non viene letto da quel file a meno che `CONFIG_FILE` non punti allo stesso percorso |
| Autenticazione tra servizi | — / `WARDEN_HMAC_KEYS`, `WARDEN_HMAC_TIMESTAMP_TOLERANCE`, `WARDEN_TLS_*` | **Solo variabili d'ambiente** (nessuna chiave YAML) |
| Integrità | — / `SNAPSHOT_MAX_AGE` | Età massima accettata dello snapshot; durata Go, predefinito `max(30s, 3 × intervallo attività)` |

## Modalità di esecuzione (MERGE_MODE)

Il sistema supporta 7 modalità di unione dei dati, selezionabili con `MERGE_MODE` (`MODE` è deprecato):

| Modalità | Descrizione | Caso d'uso |
|------|-------------|----------|
| `DEFAULT` | Comportamento storico tollerante, con priorità al remoto | Compatibilità con distribuzioni che non hanno mai scelto una modalità |
| `REMOTE_FIRST` | Il remoto prevale quando il caricamento riesce; un errore remoto è fatale e viene mantenuto l'ultimo snapshot valido noto | Distribuzioni rigorose in cui il remoto è autorevole |
| `ONLY_REMOTE` | Utilizzare solo l'origine dati remota | Dipendenza totale dalla configurazione remota |
| `ONLY_LOCAL` | Utilizzare solo il file di configurazione locale, **Redis disabilitato per impostazione predefinita** (viene abilitato se l'indirizzo `REDIS` è impostato esplicitamente o se `REDIS_ENABLED=true`) | Ambiente offline o di test |
| `LOCAL_FIRST` | Priorità al locale; i dati remoti integrano quando quelli locali non esistono | Configurazione locale come fonte principale, remoto come complemento |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Priorità al remoto, con ripiego sul locale consentito in caso di errore remoto | Scenari ad alta disponibilità |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Priorità al locale, con ripiego sul remoto consentito in caso di errore locale | Modalità ibrida |

### Freschezza degli snapshot ed errori remoti

`REMOTE_FIRST` e `ONLY_REMOTE` sono modalità rigorose. Se un aggiornamento remoto
pianificato fallisce, Warden continua a servire l'ultimo snapshot valido noto in memoria,
registra l'errore di aggiornamento e non fa avanzare il campo `loaded_at` dello snapshot.
L'endpoint di integrità restituisce HTTP 503 quando lo snapshot supera `SNAPSHOT_MAX_AGE`
(predefinito `max(30s, 3 × intervallo attività)`). Impostare il valore con una durata Go
come `2m`.

`REMOTE_FIRST_ALLOW_REMOTE_FAILED` ripiega esplicitamente su dati locali convalidati,
contrassegna lo snapshot come `degraded` e resta utilizzabile con HTTP 200. `LOCAL_FIRST`
e `LOCAL_FIRST_ALLOW_REMOTE_FAILED` possono restare integri quando la loro fonte locale
principale riesce, anche se il complemento remoto non è disponibile. `DEFAULT` mantiene il
comportamento tollerante storico per compatibilità (inclusa questa semantica di successo
locale in chiaro); per le nuove distribuzioni di produzione scegliere una modalità
esplicita. Gli errori remoti cifrati che ripiegano su dati locali vengono segnalati come
`degraded` in tutte le modalità tolleranti.

Nelle distribuzioni con più repliche, ogni replica aggiorna la propria cache e il proprio
snapshot locali al processo. Un lock Redis distribuito elegge soltanto lo scrittore della
cache Redis condivisa, per cui anche le repliche non scrittrici continuano a far avanzare
la freschezza del proprio snapshot.

### Metodi di configurazione

È possibile impostare la modalità di esecuzione nei modi seguenti:

**Argomenti da riga di comando**:
```bash
go run . --mode DEFAULT
```

**Variabili d'ambiente**:
```bash
export MERGE_MODE=DEFAULT
# MODE resta un alias di compatibilità deprecato.
```

**File di configurazione**:
```yaml
remote:
  mode: "DEFAULT"
# oppure
app:
  mode: "DEFAULT"
```

## Formato dei file di configurazione

### File locale dei dati utente (`data.json`)

Formato del file locale dei dati utente `data.json` (vedere `data.example.json`):

**Formato minimo** (solo campi obbligatori):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Formato completo** (con tutti i campi facoltativi):
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

**Descrizione dei campi**:
- `phone` (obbligatorio): numero di telefono dell'utente
- `mail` (obbligatorio): indirizzo e-mail dell'utente
- `user_id` (facoltativo): identificatore univoco dell'utente, generato automaticamente da `phone` o `mail` se non fornito
- `status` (facoltativo): stato dell'utente; i valori omessi ripiegano in sicurezza su `"inactive"`. Impostare esplicitamente `"active"` per consentire l'accesso.
- `scope` (facoltativo): array degli ambiti di autorizzazione dell'utente, per impostazione predefinita un array vuoto
- `role` (facoltativo): ruolo dell'utente, per impostazione predefinita una stringa vuota

### File di configurazione dell'applicazione (`config.yaml`)

Sono supportati i file di configurazione in formato YAML, specificati tramite il parametro `--config-file`:

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
  password: ""  # Si consiglia la variabile d'ambiente REDIS_PASSWORD o REDIS_PASSWORD_FILE
  password_file: ""  # Percorso del file della password (priorità più alta di password)
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60  # Richieste al minuto
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # Solo per lo sviluppo
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # Decifrare la risposta remota con RSA (da usare con rsa_private_key_file o REMOTE_RSA_PRIVATE_KEY)
  rsa_private_key_file: ""    # Percorso del file PEM (oppure variabile REMOTE_RSA_PRIVATE_KEY per un PEM in linea)

task:
  interval: 5s

app:
  mode: "DEFAULT"  # Modalità di unione dei dati; la politica di produzione è scelta da ENVIRONMENT
  api_key: ""      # Si consiglia la variabile d'ambiente API_KEY
  data_file: "./data.json"
  data_dir: ""     # Facoltativo: unire tutti i *.json della directory (utilizzabile insieme a data_file)
  response_fields: []  # Facoltativo: lista di campi consentiti nella risposta dell'API; vuoto = tutti i campi

tracing:
  enabled: false
  endpoint: ""     # ad es. "http://localhost:4318"
```

**Priorità di configurazione**: argomenti da riga di comando > variabili d'ambiente > file di configurazione > valori predefiniti.

**Nota sul tracciamento**: con `--config-file`, il programma principale non legge la sezione `tracing` da quel file, a meno che la variabile d'ambiente `CONFIG_FILE` non punti allo stesso percorso, oppure si usino `OTLP_ENABLED` + `OTLP_ENDPOINT`.

Vedere il file di esempio: [config.example.yaml](../../config.example.yaml).

## Argomenti da riga di comando

```bash
go run . \
  --port 8081 \                    # Porta del servizio web (predefinita: 8081)
  --redis localhost:6379 \         # Indirizzo Redis (predefinito: localhost:6379)
  --redis-password "password" \    # Password Redis (facoltativa, si consigliano le variabili d'ambiente)
  --redis-enabled=true \           # Abilitare/disabilitare Redis (predefinito: true)
  --config http://example.com/api \ # URL della configurazione remota
  --key "Bearer token" \           # Intestazione di autenticazione della configurazione remota
  --interval 5 \                   # Intervallo dell'attività pianificata (secondi, predefinito: 5)
  --mode DEFAULT \                 # Modalità di esecuzione (vedere la descrizione sopra)
  --http-timeout 5 \               # Timeout delle richieste HTTP (secondi, predefinito: 5)
  --http-max-idle-conns 100 \     # Numero massimo di connessioni HTTP inattive (predefinito: 100)
  --http-insecure-tls \           # Saltare la verifica dei certificati TLS (solo per lo sviluppo)
  --api-key "your-secret-api-key" \ # Chiave API per l'autenticazione (facoltativa, si consigliano le variabili d'ambiente)
  --config-file config.yaml        # Percorso del file di configurazione (supporta il formato YAML)
```

**Note**:
- Supporto dei file di configurazione: il parametro `--config-file` consente di specificare un file di configurazione in formato YAML
- Sicurezza della password Redis: si consiglia di usare le variabili d'ambiente `REDIS_PASSWORD` o `REDIS_PASSWORD_FILE` invece degli argomenti da riga di comando
- Verifica dei certificati TLS: `--http-insecure-tls` è destinato esclusivamente agli ambienti di sviluppo e non deve essere usato in produzione

## Variabili d'ambiente

È supportata la configurazione tramite variabili d'ambiente, con priorità inferiore rispetto agli argomenti da riga di comando. Per la tabella completa delle opzioni (incluse le regole di convalida), consultare la [CONFIGURATION zhCN](../zhCN/CONFIGURATION.md).

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Password Redis (facoltativa)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Percorso del file della password Redis (facoltativo; priorità: REDIS_PASSWORD > REDIS_PASSWORD_FILE > configurazione)
export REDIS_ENABLED=true               # Abilitare/disabilitare Redis (facoltativo, predefinito: true, accetta true/false/1/0)
                                        # Nota: in modalità ONLY_LOCAL il valore predefinito è false
                                        #       Tuttavia, se l'indirizzo REDIS è impostato esplicitamente, Redis viene abilitato automaticamente
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MERGE_MODE=DEFAULT
export DATA_FILE=./data.json          # Percorso del file locale dei dati utente
export DATA_DIR=                      # Facoltativo: directory in cui unire tutti i *.json (utilizzabile insieme a DATA_FILE)
export RESPONSE_FIELDS=               # Facoltativo: lista di campi consentiti nella risposta dell'API (separati da virgole, ad es. phone,mail,user_id,status,name); vuoto = tutti
export REMOTE_DECRYPT_ENABLED=false   # Facoltativo: decifrare la risposta remota con RSA
export REMOTE_RSA_PRIVATE_KEY_FILE=   # Facoltativo: percorso del PEM della chiave privata RSA (oppure REMOTE_RSA_PRIVATE_KEY per un PEM in linea)
export REMOTE_RSA_PRIVATE_KEY=        # Facoltativo: PEM della chiave privata RSA in linea (usato se REMOTE_RSA_PRIVATE_KEY_FILE non è impostato)
export HTTP_TIMEOUT=5                  # Timeout delle richieste HTTP (secondi)
export HTTP_MAX_IDLE_CONNS=100         # Numero massimo di connessioni HTTP inattive
export HTTP_INSECURE_TLS=false         # Se saltare la verifica dei certificati TLS (true/false oppure 1/0)
export API_KEY="your-secret-api-key"   # Chiave API per l'autenticazione (fortemente consigliata)
export CONFIG_FILE=config.yaml         # Facoltativo; carica il tracciamento dal YAML senza `--config-file`, oppure abilita il tracciamento dallo stesso file di `--config-file`
export OTLP_ENABLED=false              # Abilitare OpenTelemetry (true/false oppure 1/0)
export OTLP_ENDPOINT=http://localhost:4318  # Endpoint OTLP (obbligatorio se OTLP_ENABLED è true)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # Elenco degli IP dei proxy attendibili (separati da virgole)
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # Lista di autorizzazione IP per l'endpoint di integrità (facoltativa)
export IP_WHITELIST="192.168.1.0/24"  # Lista di autorizzazione IP globale (facoltativa)
export LOG_LEVEL="info"                # Livello di log (facoltativo, predefinito: info; opzioni: trace, debug, info, warn, error, fatal, panic)
export WARDEN_HMAC_KEYS='{"key-id":"0123456789abcdef0123456789abcdef"}'  # I segreti di produzione richiedono almeno 32 byte
export WARDEN_HMAC_ALLOW_V1=false                # Predefinito: false; impostare true solo durante una migrazione v1 di durata limitata
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # Tolleranza del timestamp HMAC (secondi)
export WARDEN_TLS_CERT=/path/to/warden.crt    # Autenticazione tra servizi: certificato TLS del server (abilita TLS insieme a KEY)
export WARDEN_TLS_KEY=/path/to/warden.key     # Chiave privata TLS del server
export WARDEN_TLS_CA=/path/to/ca.crt          # CA client (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # Richiedere il certificato client (mTLS)
```

**Priorità delle variabili d'ambiente**:
- Password Redis: `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > argomento da riga di comando `--redis-password`

**Note sulla configurazione di sicurezza**:
- `API_KEY`: protegge gli endpoint sensibili (`/`, `/log/level`); fortemente consigliato negli ambienti di produzione
- `TRUSTED_PROXY_IPS`: configurare gli IP dei reverse proxy attendibili per ottenere correttamente l'IP reale del client
- `HEALTH_CHECK_IP_WHITELIST`: limita gli IP di accesso all'endpoint di integrità (facoltativo, supporta intervalli CIDR)
- `IP_WHITELIST`: lista di autorizzazione IP globale (facoltativa, supporta intervalli CIDR)

## Requisiti dell'API di configurazione remota

L'API di configurazione remota deve restituire un array JSON nello stesso formato e può facoltativamente supportare l'autenticazione tramite intestazione Authorization.

Il formato di risposta dell'API deve corrispondere a quello del file `data.json`:

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

Se la variabile d'ambiente `KEY` o il parametro `--key` è configurato, alle richieste viene aggiunta automaticamente l'intestazione `Authorization`:

```http
Authorization: Bearer your-token-here
```

## Configurazione facoltativa dell'integrazione dei servizi

Se si sceglie di integrarsi con altri servizi (come Stargate), è possibile configurare l'autenticazione tra servizi. Di seguito le voci di configurazione pertinenti:

**Nota**: se Warden viene usato in modo autonomo, le configurazioni seguenti sono facoltative.

### Configurazione mTLS (consigliata)

Usare certificati TLS reciproci per l'autenticazione tra servizi. **Sono supportate solo le variabili d'ambiente** (nessuna chiave YAML nella configurazione dell'applicazione):

```bash
# Certificato del server Warden
export WARDEN_TLS_CERT=/path/to/warden.crt
export WARDEN_TLS_KEY=/path/to/warden.key
export WARDEN_TLS_CA=/path/to/ca.crt

# Richiedere il certificato client (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
```

### Configurazione della firma HMAC

Usare la firma HMAC-SHA256 per l'autenticazione tra servizi. **Sono supportate solo le variabili d'ambiente** (nessuna chiave YAML):

```bash
# Chiavi HMAC (formato JSON, supporta più chiavi per la rotazione)
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef","key-id-2":"abcdef0123456789abcdef0123456789"}'

# Tolleranza del timestamp (secondi), predefinita 60 quando sono impostate chiavi HMAC
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60

# La versione v1 legacy è disabilitata per impostazione predefinita. Abilitarla solo durante la migrazione dei vecchi chiamanti.
export WARDEN_HMAC_ALLOW_V1=false
```

### Configurazione delle chiamate da Stargate

Stargate deve configurare l'indirizzo del servizio Warden e le informazioni di autenticazione:

**Esempio di configurazione Stargate** (variabili d'ambiente):
```bash
# Indirizzo del servizio Warden
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# Metodo di autenticazione tra servizi (mTLS oppure HMAC)
export STARGATE_WARDEN_AUTH_TYPE=hmac

# Configurazione HMAC (se si usa HMAC)
export STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
export STARGATE_WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef

# Configurazione mTLS (se si usa mTLS)
export STARGATE_WARDEN_TLS_CERT=/path/to/stargate.crt
export STARGATE_WARDEN_TLS_KEY=/path/to/stargate.key
export STARGATE_WARDEN_TLS_CA=/path/to/ca.crt
```

### Priorità di configurazione

1. **mTLS**: se sono configurati certificati TLS, viene usato per primo mTLS
2. **HMAC**: se mTLS non è configurato, viene usata la firma HMAC
3. **Chiave API**: se nessuno dei due è configurato, si ripiega sull'autenticazione con chiave API (sconsigliata per le chiamate tra servizi)

### Convalida della configurazione

All'avvio, Warden verifica la configurazione dell'autenticazione tra servizi:

- Rifiuta una configurazione TLS parziale: certificato e chiave devono essere impostati insieme; mTLS richiede inoltre una CA client
- Se HMAC è configurato, verifica che il formato delle chiavi sia corretto
- Con `ENVIRONMENT=production`, rifiuta l'avvio se non è configurata l'autenticazione con chiave API, HMAC o mTLS

## Documentazione dettagliata della configurazione

Per informazioni più dettagliate sui meccanismi di analisi dei parametri, sulle regole di priorità e su esempi d'uso, consultare:

- [Documento di progettazione dell'analisi dei parametri](CONFIG_PARSING.md) – Documentazione dettagliata del meccanismo di analisi dei parametri
- [Documento di progettazione dell'architettura](ARCHITECTURE.md) – Comprendere l'architettura complessiva e l'impatto della configurazione
- [Documentazione di sicurezza](SECURITY.md) – Dettagli sull'autenticazione tra servizi
