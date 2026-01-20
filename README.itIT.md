# Warden

> 🌐 **Language / 语言**: [English](README.en.md) | [中文](README.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

Un servizio dati utente ad alta prestazione per liste di autorizzazione (AllowList) che supporta la sincronizzazione e la fusione di dati da fonti di configurazione locali e remote.

![Warden](.github/assets/banner.jpg)

> **Warden** (Il Guardiano) — Il guardiano della Porta Stellare che decide chi può passare e chi sarà rifiutato. Proprio come il Guardiano di Stargate protegge la Porta Stellare, Warden protegge la tua lista di autorizzazione, garantendo che solo gli utenti autorizzati possano passare.

## 📋 Panoramica del Progetto

Warden è un servizio API HTTP leggero sviluppato in Go, utilizzato principalmente per fornire e gestire dati utente di liste di autorizzazione (numeri di telefono e indirizzi email). Il servizio supporta il recupero di dati da file di configurazione locali e API remote, e fornisce multiple strategie di fusione dati per garantire prestazioni e affidabilità dei dati in tempo reale.

## ✨ Caratteristiche Principali

- 🚀 **Alte Prestazioni**: Supporta oltre 5000 richieste al secondo con una latenza media di 21ms
- 🔄 **Fonti Dati Multiple**: Supporta sia file di configurazione locali che API remote
- 🎯 **Strategie Flessibili**: Fornisce 6 modalità di fusione dati (priorità remota, priorità locale, solo remoto, solo locale, ecc.)
- ⏰ **Aggiornamenti Programmati**: Attività programmate basate su blocchi distribuiti Redis per la sincronizzazione automatica dei dati
- 📦 **Distribuzione Containerizzata**: Supporto Docker completo, pronto all'uso
- 📊 **Registrazione Strutturata**: Utilizza zerolog per fornire log di accesso e di errore dettagliati
- 🔒 **Blocchi Distribuiti**: Utilizza Redis per garantire che le attività programmate non vengano eseguite ripetutamente in ambienti distribuiti
- 🌐 **Supporto Multi-lingua**: Supporta 7 lingue (Inglese, Cinese, Francese, Italiano, Giapponese, Tedesco, Coreano) con rilevamento automatico della preferenza linguistica

## 🏗️ Progettazione dell'Architettura

Warden utilizza una progettazione architetturale a strati, inclusi lo strato HTTP, lo strato business e lo strato infrastrutturale. Il sistema supporta multiple fonti dati, cache multi-livello e meccanismi di blocco distribuiti.

Per la documentazione dettagliata dell'architettura, si prega di fare riferimento a: [Documentazione di Progettazione dell'Architettura](docs/enUS/ARCHITECTURE.md)

## 📦 Installazione ed Esecuzione

> 💡 **Guida Rapida**: Vuoi provare rapidamente Warden? Controlla i nostri [Esempi di Guida Rapida](example/README.en.md):
> - [Esempio Semplice](example/basic/README.en.md) - Utilizzo di base, solo file dati locale
> - [Esempio Avanzato](example/advanced/README.en.md) - Funzionalità complete, inclusi API remota e servizio Mock

### Prerequisiti

- Go 1.25+ (fare riferimento a [go.mod](go.mod))
- Redis (per blocchi distribuiti e cache)
- Docker (opzionale, per distribuzione containerizzata)

### Guida Rapida

1. **Clonare il progetto**
```bash
git clone <repository-url>
cd warden
```

2. **Installare le dipendenze**
```bash
go mod download
```

3. **Configurare il file dati locale**
Crea un file `data.json` (fare riferimento a `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

4. **Eseguire il servizio**
```bash
go run main.go
```

Per istruzioni dettagliate su configurazione e distribuzione, si prega di fare riferimento a:
- [Documentazione di Configurazione](docs/enUS/CONFIGURATION.md) - Scopri tutte le opzioni di configurazione
- [Documentazione di Distribuzione](docs/enUS/DEPLOYMENT.md) - Scopri i metodi di distribuzione

## ⚙️ Configurazione

Warden supporta multiple modalità di configurazione: argomenti da riga di comando, variabili d'ambiente e file di configurazione. Il sistema fornisce 6 modalità di fusione dati con strategie di configurazione flessibili.

Per la documentazione dettagliata sulla configurazione, si prega di fare riferimento a: [Documentazione di Configurazione](docs/enUS/CONFIGURATION.md)

## 📡 Documentazione API

Warden fornisce un'API RESTful completa con supporto per query di liste utente, paginazione, controlli di salute, ecc. Il progetto fornisce anche documentazione di specifica OpenAPI 3.0.

Per la documentazione API dettagliata, si prega di fare riferimento a: [Documentazione API](docs/enUS/API.md)

File di specifica OpenAPI: [openapi.yaml](openapi.yaml)

## 🌐 Supporto Multi-lingua

Warden supporta una funzionalità completa di internazionalizzazione (i18N). Tutte le risposte API, messaggi di errore e log supportano l'internazionalizzazione.

### Lingue Supportate

- 🇺🇸 Inglese (en) - Predefinito
- 🇨🇳 Cinese (zh)
- 🇫🇷 Francese (fr)
- 🇮🇹 Italiano (it)
- 🇯🇵 Giapponese (ja)
- 🇩🇪 Tedesco (de)
- 🇰🇷 Coreano (ko)

### Rilevamento della Lingua

Warden supporta due metodi di rilevamento della lingua con la seguente priorità:

1. **Parametro di query**: Specificare la lingua tramite `?lang=it`
2. **Intestazione Accept-Language**: Rilevamento automatico della preferenza linguistica del browser
3. **Lingua predefinita**: Inglese se non specificato

### Esempi di Utilizzo

```bash
# Specificare l'italiano tramite il parametro di query
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=it"

# Rilevamento automatico tramite l'intestazione Accept-Language
curl -H "X-API-Key: your-key" -H "Accept-Language: it-IT,it;q=0.9" "http://localhost:8081/"

# Utilizzare il francese
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"
```

## 🔌 Utilizzo SDK

Warden fornisce un SDK Go per facilitare l'integrazione in altri progetti. L'SDK fornisce interfacce API semplici con supporto per cache, autenticazione, ecc.

Per la documentazione SDK dettagliata, si prega di fare riferimento a: [Documentazione SDK](docs/enUS/SDK.md)

## 🐳 Distribuzione Docker

Warden supporta la distribuzione Docker e Docker Compose completa, pronto all'uso.

### Guida Rapida con Immagine Pre-costruita (Consigliato)

Usa l'immagine pre-costruita fornita da GitHub Container Registry (GHCR) per iniziare rapidamente senza costruzione locale:

```bash
# Scaricare l'immagine dell'ultima versione
docker pull ghcr.io/soulteary/warden:latest

# Eseguire il contenitore (esempio base)
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Suggerimento**: L'utilizzo di immagini pre-costruite ti consente di iniziare rapidamente senza un ambiente di costruzione locale. Le immagini vengono aggiornate automaticamente per garantire che tu stia utilizzando l'ultima versione.

### Utilizzo di Docker Compose

> 🚀 **Distribuzione Rapida**: Controlla la [Directory degli Esempi](example/README.en.md) per esempi completi di configurazione Docker Compose

Per la documentazione dettagliata sulla distribuzione, si prega di fare riferimento a: [Documentazione di Distribuzione](docs/enUS/DEPLOYMENT.md)

## 📊 Metriche delle Prestazioni

Basato sui risultati dei test di carico wrk (test di 30 secondi, 16 thread, 100 connessioni):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Latenza Media: 21.30ms
Latenza Massima: 226.09ms
```

## 📁 Struttura del Progetto

```
warden/
├── main.go                 # Punto di ingresso del programma
├── data.example.json      # Esempio di file dati locale
├── config.example.yaml    # Esempio di file di configurazione
├── openapi.yaml           # File di specifica OpenAPI
├── go.mod                 # Definizione del modulo Go
├── docker-compose.yml     # Configurazione Docker Compose
├── LICENSE                # File di licenza
├── README.*.md            # Documenti del progetto multilingue (Cinese/Inglese/Francese/Italiano/Giapponese/Tedesco/Coreano)
├── CONTRIBUTING.*.md      # Guide di contribuzione multilingue
├── docker/
│   └── Dockerfile         # File di costruzione immagine Docker
├── docs/                  # Directory di documentazione (multilingue)
│   ├── enUS/              # Documentazione inglese
│   └── zhCN/              # Documentazione cinese
├── example/               # Esempi di guida rapida
│   ├── basic/             # Esempio semplice (solo file locale)
│   └── advanced/          # Esempio avanzato (funzionalità complete, include Mock API)
├── internal/
│   ├── cache/             # Implementazione cache e blocchi Redis
│   ├── cmd/               # Analisi argomenti da riga di comando
│   ├── config/            # Gestione configurazione
│   ├── define/            # Definizioni costanti e strutture dati
│   ├── di/                # Iniezione dipendenze
│   ├── errors/            # Gestione errori
│   ├── i18n/              # Supporto internazionalizzazione
│   ├── logger/            # Inizializzazione registrazione
│   ├── metrics/           # Raccolta metriche
│   ├── middleware/        # Middleware HTTP
│   ├── parser/            # Analizzatore dati (locale/remoto)
│   ├── router/            # Gestione route HTTP
│   ├── validator/         # Validatore
│   └── version/           # Informazioni versione
├── pkg/
│   ├── gocron/            # Utilità di pianificazione attività
│   └── warden/            # SDK Warden
├── scripts/               # Directory script
└── .github/               # Configurazione GitHub (CI/CD, modelli Issue/PR, ecc.)
```

## 🔒 Funzionalità di Sicurezza

Warden implementa multiple funzionalità di sicurezza, inclusa autenticazione API, protezione SSRF, limitazione della velocità, verifica TLS, ecc.

Per la documentazione dettagliata sulla sicurezza, si prega di fare riferimento a: [Documentazione di Sicurezza](docs/enUS/SECURITY.md)

## 🔧 Guida allo Sviluppo

> 📚 **Esempi di Riferimento**: Controlla la [Directory degli Esempi](example/README.en.md) per esempi di codice e configurazioni completi per diversi scenari di utilizzo.

Per la documentazione dettagliata sullo sviluppo, si prega di fare riferimento a: [Documentazione di Sviluppo](docs/enUS/DEVELOPMENT.md)

### Standard del Codice

Il progetto segue gli standard del codice ufficiali di Go e le migliori pratiche. Per standard dettagliati, si prega di fare riferimento a:

- [CODE_STYLE.md](docs/enUS/CODE_STYLE.md) - Guida allo stile del codice
- [CONTRIBUTING.en.md](CONTRIBUTING.en.md) - Guida al contributo

## 📄 Licenza

Vedere il file [LICENSE](LICENSE) per i dettagli.

## 🤝 Contribuire

Le segnalazioni di Issues e Pull Requests sono benvenute!

## 📞 Contatto

Per domande o suggerimenti, si prega di contattare tramite Issues.

---

**Versione**: Il programma visualizza versione, ora di costruzione e versione del codice all'avvio (tramite `warden --version` o log di avvio)
