# Documentazione di Sicurezza

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Questo documento spiega le funzionalità di sicurezza di Warden, la configurazione di sicurezza e le migliori pratiche.

> ⚠️ **Nota**: Questa documentazione è in fase di traduzione. Per la versione completa, consulta la [versione inglese](../enUS/SECURITY.md).

## Funzionalità di Sicurezza Implementate

1. **Autenticazione API**: Supporta l'autenticazione tramite chiave API per proteggere gli endpoint sensibili
2. **Protezione SSRF**: Valida rigorosamente gli URL di configurazione remoti per prevenire attacchi di falsificazione delle richieste lato server
3. **Validazione degli Input**: Valida rigorosamente tutti i parametri di input per prevenire attacchi di injection
4. **Limitazione della Velocità**: Limitazione della velocità basata su IP per prevenire attacchi DDoS
5. **Verifica TLS**: Gli ambienti di produzione applicano la verifica dei certificati TLS
6. **Gestione degli Errori**: Gli ambienti di produzione nascondono informazioni dettagliate sugli errori per prevenire la perdita di informazioni
7. **Intestazioni di Risposta di Sicurezza**: Aggiunge automaticamente intestazioni di risposta HTTP relative alla sicurezza
8. **Whitelist IP**: Supporta la configurazione della whitelist IP per gli endpoint di controllo dello stato di salute
9. **Validazione dei File di Configurazione**: Previene attacchi di directory traversal
10. **Limiti di Dimensione JSON**: Limita la dimensione del corpo della risposta JSON per prevenire attacchi di esaurimento della memoria

## Migliori Pratiche di Sicurezza

### 1. Configurazione dell'Ambiente di Produzione

**Configurazione Richiesta**:
- Deve impostare la variabile d'ambiente `API_KEY`
- Impostare `MODE=production` per abilitare la modalità produzione
- Configurare `TRUSTED_PROXY_IPS` per ottenere correttamente l'IP del client
- Usare `HEALTH_CHECK_IP_WHITELIST` per limitare l'accesso al controllo dello stato di salute

### 2. Gestione delle Informazioni Sensibili

**Pratiche Consigliate**:
- ✅ Usare variabili d'ambiente per memorizzare password e chiavi
- ✅ Usare file di password (`REDIS_PASSWORD_FILE`) per memorizzare le password Redis
- ✅ Usare segnaposto o commenti nei file di configurazione
- ✅ Assicurarsi che i permessi dei file di configurazione siano impostati correttamente (ad esempio, `chmod 600`)

### 3. Sicurezza di Rete

**Configurazione Richiesta**:
- Gli ambienti di produzione devono usare HTTPS
- Configurare le regole del firewall per limitare l'accesso
- Aggiornare regolarmente le dipendenze per correggere vulnerabilità note

## Sicurezza API

### Autenticazione tramite Chiave API

Alcuni endpoint API richiedono l'autenticazione tramite chiave API.

**Metodi di Autenticazione**:
1. **Intestazione X-API-Key**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Intestazione Authorization Bearer**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Limitazione della Velocità

Per impostazione predefinita, le richieste API sono protette dalla limitazione della velocità:
- **Limite**: 60 richieste al minuto
- **Finestra**: 1 minuto
- **Superamento**: Restituisce `429 Too Many Requests`

## Segnalazione di Vulnerabilità

Se scopri una vulnerabilità di sicurezza, segnalala tramite:

1. **GitHub Security Advisory** (Preferito)
   - Vai alla scheda [Security](https://github.com/soulteary/warden/security) nel repository
   - Clicca su "Report a vulnerability"
   - Compila il modulo di advisory di sicurezza

2. **Email** (Se GitHub Security Advisory non è disponibile)
   - Invia un'email ai maintainer del progetto
   - Includi una descrizione dettagliata della vulnerabilità

## Documentazione Correlata

- [Documentazione di Configurazione](CONFIGURATION.md) - Scopri le opzioni di configurazione relative alla sicurezza
- [Documentazione di Deployment](DEPLOYMENT.md) - Scopri le raccomandazioni per il deployment in ambiente di produzione
- [Documentazione API](API.md) - Scopri le funzionalità di sicurezza dell'API
