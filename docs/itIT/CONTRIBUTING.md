# Guida ai contributi

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Grazie per l'interesse verso il progetto Warden! Ogni forma di contributo è benvenuta.

## 📋 Indice

- [Come contribuire](#come-contribuire)
- [Configurazione dell'ambiente di sviluppo](#configurazione-dellambiente-di-sviluppo)
- [Standard di codice](#standard-di-codice)
- [Politica di traduzione](#politica-di-traduzione)
- [Standard dei commit](#standard-dei-commit)
- [Processo di pull request](#processo-di-pull-request)
- [Segnalazioni di bug e richieste di funzionalità](#segnalazioni-di-bug-e-richieste-di-funzionalità)

## 🚀 Come contribuire

È possibile contribuire nei modi seguenti:

- **Segnalare bug**: segnala i problemi nelle GitHub Issues
- **Proporre funzionalità**: proponi nuove idee nelle GitHub Issues
- **Inviare codice**: proponi miglioramenti tramite Pull Request
- **Migliorare la documentazione**: aiuta a migliorare la documentazione del progetto
- **Rispondere alle domande**: aiuta gli altri utenti nelle Issues

Partecipando a questo progetto, rispetta tutti i collaboratori, accetta le critiche costruttive e concentrati su ciò che è meglio per il progetto.

## 🛠️ Configurazione dell'ambiente di sviluppo

### Prerequisiti

- Go 1.27 o versione successiva
- Redis (per i test)
- Git

### Avvio rapido

```bash
# 1. Effettuare il fork e clonare il progetto
git clone https://github.com/your-username/warden.git
cd warden

# 2. Aggiungere il repository upstream
git remote add upstream https://github.com/soulteary/warden.git

# 3. Installare le dipendenze
go mod download

# 4. Eseguire i test
go test ./...

# 5. Avviare il servizio in locale (assicurarsi che Redis sia in esecuzione)
go run .
```

## 📝 Standard di codice

Attieniti ai seguenti standard di codice:

1. **Seguire gli standard ufficiali di Go**: [Effective Go](https://go.dev/doc/effective_go)
2. **Formattare il codice**: eseguire `go fmt ./...`
3. **Controllare il codice**: usare `golangci-lint` oppure `go vet ./...`
4. **Scrivere test**: le nuove funzionalità devono includere test
5. **Aggiungere commenti**: funzioni e tipi pubblici devono avere commenti di documentazione
6. **Denominazione delle costanti**: tutte le costanti devono usare lo stile `ALL_CAPS` (UPPER_SNAKE_CASE)

Per le linee guida dettagliate sullo stile del codice, consulta [CODE_STYLE.md](CODE_STYLE.md).

## 🌐 Politica di traduzione

Warden distribuisce documentazione e messaggi di runtime in sette lingue. **Non** sono tutte
mantenute con lo stesso livello di rigore, e fingere il contrario è esattamente il motivo per
cui cinque locale sono rimaste indietro, in silenzio, di 18 chiavi di traduzione e di diversi
documenti.

**Livelli**

| Livello | Lingue | Aspettativa |
| --- | --- | --- |
| Autorevole | Inglese (`enUS`), cinese semplificato (`zhCN`) | Aggiornate nella stessa pull request della modifica. Una PR che cambia il comportamento senza aggiornare entrambe è incompleta. |
| Al meglio possibile | `deDE`, `frFR`, `itIT`, `jaJP`, `koKR` | Possono restare indietro. Ogni documento in ritardo riporta un banner che rimanda il lettore alle versioni autorevoli. |

**Le stringhe di runtime non rientrano nel "al meglio possibile".** `locales/*.json` è
verificato da `go test ./locales/`, che fallisce quando una locale:

- manca di una chiave definita in `en.json` (o ne definisce una assente lì),
- presenta una sequenza di verbi `printf` (`%s`, `%d`) diversa da quella della sorgente inglese, oppure
- contiene un valore identico byte per byte a quello inglese (una stringa non tradotta).

Le chiavi mancanti ripiegano sull'inglese a runtime, quindi nulla si rompe in modo visibile —
ed è proprio per questo che il controllo esiste. Quando aggiungi un messaggio visibile
all'utente, inserisci la chiave in **tutti e sette** i file di locale nello stesso commit. Se
un valore è legittimamente identico all'inglese (un prestito linguistico, il nome di un
protocollo), aggiungilo a `intentionallyIdentical` in `locales/locales_test.go` con un
commento, invece di eliminare il controllo.

Esegui `make docs-parity` per vedere quanto ciascun documento tradotto si è allontanato da `enUS`.

## 📦 Standard dei commit

### Formato del messaggio di commit

Utilizziamo lo standard [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Tipi

- `feat`: nuova funzionalità
- `fix`: correzione di bug
- `docs`: aggiornamento della documentazione
- `style`: modifiche alla formattazione del codice (senza effetti sull'esecuzione)
- `refactor`: refactoring del codice
- `perf`: ottimizzazione delle prestazioni
- `test`: relativo ai test
- `chore`: modifiche al processo di build o agli strumenti ausiliari

### Esempi

```
feat(cache): Add Redis cache support

Implemented Redis-based distributed cache, supporting data persistence and multi-instance sharing.

Closes #123
```

```
fix(router): Fix pagination parameter validation issue

Fixed the issue where incorrect status code was returned when page_size exceeds maximum value.

Fixes #456
```

## 🔄 Processo di pull request

### Creare una pull request

```bash
# 1. Creare un branch per la funzionalità
git checkout -b feature/your-feature-name

# 2. Apportare le modifiche ed effettuare il commit
git add .
git commit -m "feat: Add new feature"

# 3. Sincronizzare il codice upstream
git fetch upstream
git rebase upstream/main

# 4. Effettuare il push del branch e creare la PR
git push origin feature/your-feature-name
```

### Lista di controllo per la pull request

Prima di inviare una Pull Request, assicurati che:

- [ ] Il codice rispetti gli standard di codice del progetto
- [ ] Tutti i test passino (`go test ./...`)
- [ ] Il codice sia formattato (`go fmt ./...`)
- [ ] Siano stati aggiunti i test necessari
- [ ] La documentazione correlata sia stata aggiornata
- [ ] Il messaggio di commit rispetti gli [standard dei commit](#standard-dei-commit)
- [ ] Il codice superi i controlli di lint

Tutte le Pull Request richiedono una revisione del codice. Rispondi tempestivamente ai commenti di revisione.

## 🐛 Segnalazioni di bug e richieste di funzionalità

Prima di creare una Issue, cerca tra quelle esistenti per verificare che il problema o la funzionalità non siano già stati segnalati.

### Modello di segnalazione di bug

```markdown
**Descrizione**
Descrivi il bug in modo chiaro e conciso.

**Passaggi per riprodurlo**
1. Eseguire '...'
2. Osservare l'errore

**Comportamento atteso**
Descrivi in modo chiaro e conciso ciò che ti aspettavi accadesse.

**Comportamento effettivo**
Descrivi in modo chiaro e conciso ciò che è realmente accaduto.

**Informazioni sull'ambiente**
- Sistema operativo: [ad es. macOS 12.0]
- Versione di Go: [ad es. 1.27]
- Versione di Redis: [ad es. 7.0]
```

### Modello di richiesta di funzionalità

```markdown
**Descrizione della funzionalità**
Descrivi in modo chiaro e conciso la funzionalità desiderata.

**Descrizione del problema**
Quale problema risolve questa funzionalità? Perché è necessaria?

**Soluzione proposta**
Descrivi in modo chiaro e conciso come vorresti che venisse realizzata.
```

## 🎯 Come iniziare

Se vuoi contribuire ma non sai da dove cominciare, puoi concentrarti su:

- Le Issue con l'etichetta `good first issue`
- Le Issue con l'etichetta `help wanted`
- I commenti `TODO` nel codice
- I miglioramenti alla documentazione (correggere refusi, migliorare la chiarezza, aggiungere esempi)

In caso di dubbi, consulta le Issue e le Pull Request esistenti oppure chiedi nella Issue pertinente.

---

Grazie ancora per il tuo contributo al progetto Warden! 🎉
