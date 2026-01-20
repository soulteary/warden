# Guida al Contributo

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Grazie per il tuo interesse nel progetto Warden! Accogliamo tutte le forme di contributo.

> ⚠️ **Nota**: Questa documentazione è in fase di traduzione. Per la versione completa, consulta la [versione inglese](../enUS/CONTRIBUTING.md).

## 📋 Indice

- [Come Contribuire](#come-contribuire)
- [Configurazione dell'Ambiente di Sviluppo](#configurazione-dellambiente-di-sviluppo)
- [Standard del Codice](#standard-del-codice)
- [Standard dei Commit](#standard-dei-commit)
- [Processo Pull Request](#processo-pull-request)
- [Segnalazione Bug e Richieste di Funzionalità](#segnalazione-bug-e-richieste-di-funzionalità)

## 🚀 Come Contribuire

Puoi contribuire nei seguenti modi:

- **Segnalare Bug**: Segnalare problemi in GitHub Issues
- **Suggerire Funzionalità**: Proporre nuove idee di funzionalità in GitHub Issues
- **Inviare Codice**: Inviare miglioramenti del codice tramite Pull Requests
- **Migliorare la Documentazione**: Aiutare a migliorare la documentazione del progetto
- **Rispondere alle Domande**: Aiutare altri utenti nelle Issues

Quando partecipi a questo progetto, per favore rispetta tutti i contributori, accetta critiche costruttive e concentrati su ciò che è meglio per il progetto.

## 🛠️ Configurazione dell'Ambiente di Sviluppo

### Prerequisiti

- Go 1.25 o superiore
- Redis (per i test)
- Git

### Avvio Rapido

```bash
# 1. Fork e clona il progetto
git clone https://github.com/your-username/warden.git
cd warden

# 2. Aggiungi il repository upstream
git remote add upstream https://github.com/soulteary/warden.git

# 3. Installa le dipendenze
go mod download

# 4. Esegui i test
go test ./...

# 5. Avvia il servizio locale (assicurati che Redis sia in esecuzione)
go run main.go
```

## 📝 Standard del Codice

Per favore segui questi standard del codice:

1. **Segui gli Standard del Codice Ufficiali di Go**: [Effective Go](https://go.dev/doc/effective_go)
2. **Formatta il Codice**: Esegui `go fmt ./...`
3. **Controllo del Codice**: Usa `golangci-lint` o `go vet ./...`
4. **Scrivi Test**: Le nuove funzionalità devono includere test
5. **Aggiungi Commenti**: Le funzioni e i tipi pubblici devono avere commenti di documentazione
6. **Denominazione delle Costanti**: Tutte le costanti devono usare lo stile `ALL_CAPS` (UPPER_SNAKE_CASE)

Per linee guida dettagliate sullo stile del codice, consulta [CODE_STYLE.md](CODE_STYLE.md).

## 📦 Standard dei Commit

### Formato del Messaggio di Commit

Usiamo lo standard [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Tipi

- `feat`: Nuova funzionalità
- `fix`: Correzione bug
- `docs`: Aggiornamento documentazione
- `style`: Regolazione formato codice (non influisce sull'esecuzione del codice)
- `refactor`: Refactoring del codice
- `perf`: Ottimizzazione prestazioni
- `test`: Relativo ai test
- `chore`: Modifiche al processo di build o agli strumenti ausiliari

## 🔄 Processo Pull Request

### Crea una Pull Request

```bash
# 1. Crea un branch per la funzionalità
git checkout -b feature/your-feature-name

# 2. Fai modifiche e committa
git add .
git commit -m "feat: Aggiungi nuova funzionalità"

# 3. Sincronizza il codice upstream
git fetch upstream
git rebase upstream/main

# 4. Pusha il branch e crea una PR
git push origin feature/your-feature-name
```

### Checklist Pull Request

Prima di inviare una Pull Request, assicurati che:

- [ ] Il codice segue gli standard del codice del progetto
- [ ] Tutti i test passano (`go test ./...`)
- [ ] Il codice è formattato (`go fmt ./...`)
- [ ] I test necessari sono aggiunti
- [ ] La documentazione correlata è aggiornata
- [ ] Il messaggio di commit segue gli [Standard dei Commit](#standard-dei-commit)
- [ ] Il codice supera i controlli lint

Tutte le Pull Requests richiedono una revisione del codice. Per favore rispondi prontamente ai commenti di revisione.

## 🐛 Segnalazione Bug e Richieste di Funzionalità

Prima di creare una Issue, per favore cerca le Issues esistenti per confermare che il problema o la funzionalità non siano stati segnalati.

## 🎯 Iniziare

Se vuoi contribuire ma non sai da dove iniziare, puoi concentrarti su:

- Issues etichettate `good first issue`
- Issues etichettate `help wanted`
- Commenti `TODO` nel codice
- Miglioramenti della documentazione (correggere errori di battitura, migliorare la chiarezza, aggiungere esempi)

Se hai domande, consulta le Issues e Pull Requests esistenti, o chiedi nelle Issues pertinenti.

---

Grazie ancora per il contributo al progetto Warden! 🎉
