# Indice della Documentazione

Benvenuto nella documentazione del servizio dati utente Warden AllowList.

## 🌐 Documentazione Multilingue

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 Elenco dei Documenti

### Documenti Principali

- **[README.md](../../README.itIT.md)** - Panoramica del progetto e guida rapida
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architettura tecnica e decisioni di progettazione

### Documenti Dettagliati

- **[API.md](API.md)** - Documentazione completa degli endpoint API
  - Endpoint di query della lista utenti
  - Funzionalità di paginazione
  - Endpoint di controllo dello stato
  - Formati di risposta degli errori

- **[CONFIGURATION.md](CONFIGURATION.md)** - Riferimento di configurazione
  - Metodi di configurazione
  - Elementi di configurazione richiesti
  - Elementi di configurazione opzionali
  - Strategie di unione dei dati
  - Esempi di configurazione
  - Best practice di configurazione

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Guida al deployment
  - Deployment Docker (inclusi immagini GHCR)
  - Deployment Docker Compose
  - Deployment locale
  - Deployment in ambiente di produzione
  - Deployment Kubernetes
  - Ottimizzazione delle prestazioni

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Guida allo sviluppo
  - Configurazione dell'ambiente di sviluppo
  - Spiegazione della struttura del codice
  - Guida ai test
  - Guida al contributo

- **[SDK.md](SDK.md)** - Documentazione sull'uso dell'SDK
  - Installazione e uso dell'SDK Go
  - Descrizione dell'interfaccia API
  - Codice di esempio

- **[SECURITY.md](SECURITY.md)** - Documentazione sulla sicurezza
  - Funzionalità di sicurezza
  - Configurazione della sicurezza
  - Best practice

- **[CODE_STYLE.md](CODE_STYLE.md)** - Guida allo stile del codice
  - Standard del codice
  - Convenzioni di denominazione
  - Best practice

## 🌐 Supporto Multilingue

Warden fornisce messaggi tradotti tramite cataloghi di lingue incorporati.

### Lingue Supportate

- 🇺🇸 Inglese (en) - Lingua predefinita
- 🇨🇳 Cinese (zh)
- 🇫🇷 Francese (fr)
- 🇮🇹 Italiano (it)
- 🇯🇵 Giapponese (ja)
- 🇩🇪 Tedesco (de)
- 🇰🇷 Coreano (ko)

### Rilevamento della Lingua

Warden supporta due metodi di rilevamento della lingua con la seguente priorità:

1. **Parametro di Query**: Specificare la lingua tramite il parametro di query URL `?lang=it`
2. **Header Accept-Language**: Rilevamento automatico della preferenza linguistica del browser o del client
3. **Lingua Predefinita**: Inglese se non specificato

### Esempi di Utilizzo

#### Specificare la Lingua tramite Parametro di Query

```bash
# Usare l'italiano
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=it"

# Usare il giapponese
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"

# Usare il francese
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"
```

#### Rilevamento Automatico tramite Header Accept-Language

```bash
# Il browser invia automaticamente l'header Accept-Language
curl -H "X-API-Key: your-key" \
     -H "Accept-Language: it-IT,it;q=0.9,en;q=0.8" \
     "http://localhost:8081/"
```

### Ambito di Internazionalizzazione

Il seguente contenuto supporta più lingue:

- ✅ Messaggi di risposta di errore API
- ✅ Messaggi di errore del codice di stato HTTP
- ✅ Messaggi di log (basati sul contesto della richiesta)
- ✅ Messaggi di configurazione e avviso

### Implementazione Tecnica

- Utilizza il contesto della richiesta per memorizzare le informazioni sulla lingua, evita lo stato globale
- Supporta il cambio di lingua thread-safe
- Fallback automatico all'inglese (se la traduzione non viene trovata)
- Tutte le traduzioni sono integrate nel codice, nessun file esterno richiesto

### Note di Sviluppo

Machine-readable fields, reason codes and fixed diagnostic events remain untranslated; background tasks can use a fixed language. Localization does not change protocol fields.

Modificare i cataloghi JSON incorporati in `locales/*.json`. Per una nuova lingua, verificare anche `locales/embed.go` e la registrazione in `internal/i18n/i18n.go`.

## 🚀 Navigazione Rapida

### Per Iniziare

1. Leggere [README.itIT.md](../../README.itIT.md) per comprendere il progetto
2. Controllare la sezione [Guida Rapida](../../README.itIT.md#guida-rapida)
3. Fare riferimento a [Configurazione](../../README.itIT.md#configurazione) per configurare il servizio

### Sviluppatori

1. Leggere [ARCHITECTURE.md](ARCHITECTURE.md) per comprendere l'architettura
2. Controllare [API.md](API.md) per comprendere le interfacce API
3. Fare riferimento alla [Guida allo Sviluppo](DEVELOPMENT.md) per lo sviluppo

### Operazioni

1. Leggere [DEPLOYMENT.md](DEPLOYMENT.md) per comprendere i metodi di deployment
2. Controllare [CONFIGURATION.md](CONFIGURATION.md) per comprendere le opzioni di configurazione
3. Fare riferimento a [Ottimizzazione delle Prestazioni](DEPLOYMENT.md#ottimizzazione-delle-prestazioni) per ottimizzare il servizio

## 📖 Struttura dei Documenti

```
warden/
├── README.md              # Documento principale del progetto (Italiano)
├── README.itIT.md         # Documento principale del progetto (Italiano)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Indice della documentazione (Inglese)
│   │   ├── ARCHITECTURE.md # Documento di architettura (Inglese)
│   │   ├── API.md          # Documento API (Inglese)
│   │   ├── CONFIGURATION.md # Riferimento di configurazione (Inglese)
│   │   ├── DEPLOYMENT.md   # Guida al deployment (Inglese)
│   │   ├── DEVELOPMENT.md  # Guida allo sviluppo (Inglese)
│   │   ├── SDK.md          # Documento SDK (Inglese)
│   │   ├── SECURITY.md     # Documento di sicurezza (Inglese)
│   │   └── CODE_STYLE.md   # Stile del codice (Inglese)
│   └── itIT/
│       ├── README.md       # Indice della documentazione (Italiano, questo file)
│       ├── ARCHITECTURE.md # Documento di architettura (Italiano)
│       ├── API.md          # Documento API (Italiano)
│       ├── CONFIGURATION.md # Riferimento di configurazione (Italiano)
│       ├── DEPLOYMENT.md   # Guida al deployment (Italiano)
│       ├── DEVELOPMENT.md  # Guida allo sviluppo (Italiano)
│       ├── SDK.md          # Documento SDK (Italiano)
│       ├── SECURITY.md     # Documento di sicurezza (Italiano)
│       └── CODE_STYLE.md   # Stile del codice (Italiano)
└── ...
```

## 🔍 Ricerca per Argomento

### Relativo alla Configurazione

- Configurazione delle variabili d'ambiente: [CONFIGURATION.md](CONFIGURATION.md)
- Strategie di unione dei dati: [CONFIGURATION.md](CONFIGURATION.md)
- Esempi di configurazione: [CONFIGURATION.md](CONFIGURATION.md)

### Relativo all'API

- Elenco degli endpoint API: [API.md](API.md)
- Gestione degli errori: [API.md](API.md)
- Funzionalità di paginazione: [API.md](API.md)

### Relativo al Deployment

- Deployment Docker: [DEPLOYMENT.md#deployment-docker](DEPLOYMENT.md#deployment-docker)
- Immagini GHCR: [DEPLOYMENT.md#utilizzo-di-immagini-precompilate-consigliato](DEPLOYMENT.md#utilizzo-di-immagini-precompilate-consigliato)
- Ambiente di produzione: [DEPLOYMENT.md#deployment-ambiente-di-produzione-raccomandazioni](DEPLOYMENT.md#deployment-ambiente-di-produzione-raccomandazioni)
- Kubernetes: [DEPLOYMENT.md#deployment-kubernetes](DEPLOYMENT.md#deployment-kubernetes)

### Relativo all'Architettura

- Stack tecnologico: [ARCHITECTURE.md](ARCHITECTURE.md)
- Struttura del progetto: [ARCHITECTURE.md](ARCHITECTURE.md)
- Componenti principali: [ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 Raccomandazioni d'Uso

1. **Utenti per la prima volta**: Iniziare con [README.itIT.md](../../README.itIT.md) e seguire la guida rapida
2. **Configurare il servizio**: Fare riferimento a [CONFIGURATION.md](CONFIGURATION.md) per comprendere tutte le opzioni di configurazione
3. **Deployare il servizio**: Controllare [DEPLOYMENT.md](DEPLOYMENT.md) per comprendere i metodi di deployment
4. **Sviluppare estensioni**: Leggere [ARCHITECTURE.md](ARCHITECTURE.md) per comprendere la progettazione dell'architettura
5. **Integrare l'SDK**: Fare riferimento a [SDK.md](SDK.md) per imparare come usare l'SDK

## 📝 Aggiornamenti dei Documenti

La documentazione viene aggiornata continuamente man mano che il progetto si evolve. Se trovi errori o hai bisogno di aggiunte, invia un Issue o una Pull Request.

## 🤝 Contribuire

I miglioramenti alla documentazione sono benvenuti:

1. Trovare errori o aree che necessitano di miglioramento
2. Inviare un Issue che descrive il problema
3. O inviare direttamente una Pull Request

## Release and ownership contracts

- [Release notes and upgrade requirements](../releases/v1.5.0.md)
- [Scope ownership and retained memory](../enUS/SCOPE_OWNERSHIP.md)
