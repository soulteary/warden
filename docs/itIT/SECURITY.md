# Documentazione di sicurezza

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Questo documento illustra le funzionalità di sicurezza di Warden, la relativa configurazione e le buone pratiche.

## Funzionalità di sicurezza implementate

1. **Autenticazione dell'API**: supporta l'autenticazione tramite chiave API per proteggere gli endpoint sensibili
2. **Protezione SSRF**: convalida rigorosamente gli URL di configurazione remota per prevenire attacchi di Server-Side Request Forgery
3. **Convalida degli input**: convalida rigorosamente tutti i parametri di input per prevenire attacchi di injection
4. **Limitazione di frequenza**: limitazione basata sull'indirizzo IP per prevenire attacchi DDoS
5. **Verifica TLS**: gli ambienti di produzione impongono la verifica dei certificati TLS
6. **Gestione degli errori**: gli ambienti di produzione nascondono le informazioni dettagliate sugli errori per evitare fughe di informazioni
7. **Intestazioni di risposta di sicurezza**: aggiunge automaticamente le intestazioni HTTP legate alla sicurezza
8. **Lista di autorizzazione IP**: consente di configurare una lista di autorizzazione IP per gli endpoint di controllo integrità
9. **Convalida del file di configurazione**: previene gli attacchi di path traversal
10. **Limiti di dimensione JSON**: limita la dimensione del corpo delle risposte JSON per prevenire attacchi di esaurimento della memoria
11. **Limite di lunghezza dei parametri di query utente**: un singolo parametro (`phone`/`mail`/`user_id`) non deve superare 512 byte, per prevenire DoS e la crescita incontrollata di log e cache
12. **Sanificazione dei dati personali nei log di audit**: l'identificatore scritto nell'audit viene mascherato per phone/mail, così da evitare l'esposizione di dati personali qualora l'archivio di audit venga compromesso

## Buone pratiche di sicurezza

### 1. Configurazione dell'ambiente di produzione

**Configurazione obbligatoria**:
- **Occorre** impostare `ENVIRONMENT=production` per attivare l'irrobustimento di produzione.
- **Occorre** configurare almeno un meccanismo di autenticazione del servizio: `API_KEY`, HMAC v2 oppure mTLS.
- **Occorre** configurare `TRUSTED_PROXY_IPS` per ottenere correttamente l'IP del client
- **Occorre** usare `HEALTH_CHECK_IP_WHITELIST` per limitare l'accesso al controllo di integrità (oppure limitare `/health` e `/healthcheck` a livello di rete o reverse proxy)
- **Occorre** limitare `/metrics`. Con `ENVIRONMENT=production` l'autenticazione è richiesta per impostazione predefinita; mantenere questo comportamento (oppure limitare il percorso a livello di reverse proxy o di rete) e **non** impostare `WARDEN_METRICS_REQUIRE_AUTH=false` in produzione.

**Esempio di configurazione**:
```bash
export API_KEY="your-strong-api-key-here"
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. Gestione delle informazioni sensibili

**Pratiche consigliate**:
- ✅ Conservare password e chiavi in variabili d'ambiente
- ✅ Usare file di password (`REDIS_PASSWORD_FILE`) per le password di Redis
- ✅ Usare segnaposto o commenti nei file di configurazione
- ✅ Verificare che i permessi dei file di configurazione siano corretti (ad es. `chmod 600`)

**Sconsigliato**:
- ❌ Inserire le password direttamente nei file di configurazione
- ❌ Passare le password tramite argomenti da riga di comando (compaiono nell'elenco dei processi)
- ❌ Inserire nel controllo di versione file di configurazione contenenti informazioni sensibili

**Esempio**:
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # Usare la variabile d'ambiente REDIS_PASSWORD oppure REDIS_PASSWORD_FILE

app:
  # api_key: ""  # Usare la variabile d'ambiente API_KEY
```

### 3. Sicurezza di rete

**Configurazione obbligatoria**:
- Gli ambienti di produzione devono usare HTTPS
- Configurare regole di firewall per limitare l'accesso
- Aggiornare regolarmente le dipendenze per correggere le vulnerabilità note

**Configurazione consigliata**:
- Usare un reverse proxy (come Nginx) per gestire SSL/TLS
- Configurare `TRUSTED_PROXY_IPS` per ottenere correttamente l'IP reale del client
- Usare password e chiavi API robuste
- Disattivare `HTTP_INSECURE_TLS` (in produzione deve valere `false`)

### 4. Monitoraggio e audit

**Pratiche consigliate**:
- Monitorare i log degli eventi di sicurezza
- Esaminare regolarmente i log di accesso
- Usare strumenti di analisi della sicurezza nella CI/CD
- Predisporre meccanismi di allerta

**Gestione del livello di log**:
- In produzione si consiglia il livello `info` oppure `warn`
- Tutte le operazioni di modifica del livello di log sono registrate nei log di audit di sicurezza
- Il livello di log può essere modificato dinamicamente tramite l'API `/log/level` (richiede autenticazione con chiave API)

## Sicurezza dell'API

### Autenticazione tramite chiave API

Alcuni endpoint richiedono l'autenticazione tramite chiave API:

**Endpoint che richiedono autenticazione**:
- `GET /` – Ottenere l'elenco degli utenti
- `GET /user` – Interrogare un singolo utente
- `GET /log/level` – Ottenere il livello di log
- `POST /log/level` – Impostare il livello di log

**Endpoint senza autenticazione** (in produzione devono essere protetti in altro modo):
- `GET /health` – Controllo di integrità (**occorre** configurare `HEALTH_CHECK_IP_WHITELIST` oppure un isolamento di rete)
- `GET /healthcheck` – Controllo di integrità (come sopra)
- `GET /metrics` – Metriche Prometheus (**occorre** impostare una chiave API per lo scraping oppure limitare l'accesso tramite reverse proxy o rete; non esporre pubblicamente)

**Metodi di autenticazione**:
1. **Intestazione X-API-Key**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Intestazione Authorization Bearer**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Limitazione di frequenza

Per impostazione predefinita, le richieste all'API sono protette da una limitazione di frequenza:

- **Limite**: 60 richieste al minuto
- **Finestra**: 1 minuto
- **In caso di superamento**: restituisce `429 Too Many Requests`

Può essere modificata tramite il file di configurazione:

```yaml
rate_limit:
  rate: 60  # Richieste al minuto
  window: 1m
```

### Lista di autorizzazione IP

Sono supportati due tipi di lista di autorizzazione IP:

1. **Lista di autorizzazione IP globale** (`IP_WHITELIST`):
   - Limita l'accesso a tutti gli endpoint
   - Supporta il formato di intervallo CIDR

2. **Lista di autorizzazione IP del controllo di integrità** (`HEALTH_CHECK_IP_WHITELIST`):
   - Limita soltanto gli endpoint `/health` e `/healthcheck`
   - Supporta il formato di intervallo CIDR

**Esempio di configurazione**:
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Sicurezza dei dati

### Sicurezza dell'API di configurazione remota

- Le API di configurazione remota dovrebbero usare meccanismi di autenticazione (intestazione Authorization)
- Si consiglia il protocollo HTTPS
- Verificare i certificati TLS dell'API remota (obbligatorio in produzione)

### Sicurezza di Redis

- Redis dovrebbe essere configurato con protezione tramite password
- Usare le variabili d'ambiente `REDIS_PASSWORD` oppure `REDIS_PASSWORD_FILE`
- Limitare l'accesso di rete a Redis (consentirlo solo al server applicativo)
- Aggiornare regolarmente Redis per correggere le vulnerabilità note

### Sicurezza del file di dati

- Verificare che i permessi del file `data.json` siano corretti
- Non inserire dati sensibili nel controllo di versione
- Effettuare backup regolari dei file di dati

## Intestazioni di risposta di sicurezza

Warden aggiunge automaticamente le seguenti intestazioni HTTP legate alla sicurezza:

- `X-Content-Type-Options: nosniff` – Impedisce il MIME type sniffing
- `X-Frame-Options: DENY` – Impedisce il clickjacking
- `X-XSS-Protection: 1; mode=block` – Protezione XSS

## Gestione degli errori

### Modalità produzione

In modalità produzione (`ENVIRONMENT=production`):

- Le informazioni dettagliate sugli errori vengono nascoste per evitare fughe di informazioni
- Vengono restituiti messaggi di errore generici
- Le informazioni dettagliate sugli errori sono registrate solo nei log

### Modalità sviluppo

In modalità sviluppo:

- Vengono mostrate informazioni dettagliate sugli errori per facilitare il debug
- Sono incluse le informazioni sullo stack trace

## Audit di sicurezza

Per le indicazioni su irrobustimento e verifica delle release, vedere [Release Security](../RELEASE_SECURITY.md).

## Segnalazione di vulnerabilità

Se individui una vulnerabilità di sicurezza, segnalala così:

1. Crea una Issue di sicurezza privata (se supportato)
2. Invia un'e-mail ai manutentori del progetto
3. Non divulgare pubblicamente la vulnerabilità finché non è stata corretta

## Autenticazione tra servizi (facoltativa)

Se scegli di integrarti con altri servizi (come Stargate), l'autenticazione tra servizi consente di garantire la sicurezza. **mTLS e HMAC sono implementati**; l'ordine di priorità è **mTLS > HMAC > chiave API**. Warden supporta i metodi seguenti:

**Nota**: se Warden viene usato in modo autonomo, l'autenticazione tra servizi è facoltativa.

### mTLS (consigliato)

Usare certificati TLS reciproci per l'autenticazione, ottenendo un livello di sicurezza superiore.

**Configurazione**:

1. **Generare i certificati**:
   ```bash
   # Generare il certificato della CA
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # Generare il certificato del server Warden
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # Generare il certificato client di Stargate
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Configurazione di Warden** (variabili d'ambiente):
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Configurazione di Stargate**:
   - Configurare il percorso del certificato client
   - Configurare il percorso del certificato della CA per verificare il certificato del server Warden

### Firma HMAC

Usare la firma HMAC-SHA256 per verificare le richieste; è più semplice da mettere in opera.

**Algoritmo di firma**:
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**Intestazioni della richiesta**:
- `X-Signature`: valore della firma HMAC
- `X-Timestamp`: timestamp Unix (secondi)
- `X-Key-Id`: ID della chiave (incluso nella firma, per una rotazione sicura delle chiavi)
- `X-Nonce`: nonce univoco a 128 bit in esadecimale
- `X-Signature-Version`: `v2`

**Configurazione di Warden** (variabili d'ambiente):
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # Tolleranza del timestamp (secondi), predefinita 60
export WARDEN_HMAC_ALLOW_V1=false          # Valore predefinito; impostare true solo durante una migrazione legacy di durata limitata
```

La configurazione di produzione richiede che ogni segreto HMAC contenga almeno 32 byte grezzi.
Genera i segreti con una sorgente casuale crittograficamente sicura e conservali in un
gestore di segreti; non riutilizzare il valore illustrativo riportato sopra.

**Esempio con l'SDK Go**:
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**Regole di verifica**:
- Warden verifica che il timestamp rientri nell'intervallo di tolleranza (predefinito ±60 secondi)
- Warden verifica tutti i campi canonici, incluso l'ID della chiave, e rifiuta i nonce riutilizzati
- Se la verifica della firma fallisce, restituisce `401 Unauthorized`

### Priorità di configurazione

1. **mTLS**: se sono configurati certificati TLS, viene usato per primo mTLS
2. **HMAC**: se mTLS non è configurato, viene usata la firma HMAC
3. **Chiave API**: se nessuno dei due è configurato, si ripiega sull'autenticazione con chiave API (sconsigliata per le chiamate tra servizi)

### Raccomandazioni di sicurezza

1. **Ambiente di produzione**: si consiglia vivamente di usare mTLS per l'autenticazione tra servizi
2. **Gestione delle chiavi**: usa servizi di gestione delle chiavi (come HashiCorp Vault) per conservare chiavi e certificati
3. **Rotazione delle chiavi**: ruota regolarmente le chiavi HMAC e i certificati TLS
4. **Isolamento di rete**: quando possibile, usa policy di rete per consentire l'accesso a Warden solo da Stargate

## Documentazione correlata

- [Documentazione di configurazione](CONFIGURATION.md) – Scopri le opzioni di configurazione legate alla sicurezza
- [Documentazione di distribuzione](DEPLOYMENT.md) – Scopri i consigli per la distribuzione in produzione
- [Documentazione dell'API](API.md) – Scopri le funzionalità di sicurezza dell'API
- [Documentazione di architettura](ARCHITECTURE.md) – Scopri l'architettura di integrazione dei servizi
