# Warden OpenTelemetry Tracing

Il servizio Warden supporta il tracciamento distribuito OpenTelemetry per monitorare e debuggare le catene di chiamate tra servizi.

## Funzionalità

- **Tracciamento automatico delle richieste HTTP**: Crea automaticamente span per tutte le richieste HTTP
- **Tracciamento delle query utente**: Aggiunge informazioni di tracciamento dettagliate per l'endpoint `/user`
- **Propagazione del contesto**: Supporta lo standard W3C Trace Context, si integra perfettamente con i servizi Stargate e Herald
- **Configurabile**: Abilita/disabilita tramite variabili d'ambiente o file di configurazione

## Configurazione

Warden supporta due metodi per configurare il tracciamento OpenTelemetry:

1. **File di configurazione** (consigliato per la produzione)
2. **Variabili d'ambiente** (comodo per lo sviluppo)

**Priorità**: Il file di configurazione ha la precedenza sulle variabili d'ambiente.

### Metodo 1: File di configurazione (YAML)

Crea un file di configurazione (ad esempio `config.yaml`) e specificarlo tramite la variabile d'ambiente `CONFIG_FILE`:

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

**Utilizzo**:
```bash
export CONFIG_FILE=/path/to/config.yaml
./warden
```

**Vantaggi**:
- Gestione centralizzata della configurazione
- Migliore per gli ambienti di produzione
- Supporta tutte le opzioni di configurazione in un unico file

### Metodo 2: Variabili d'ambiente

```bash
# Abilita il tracciamento OpenTelemetry
OTLP_ENABLED=true

# Endpoint OTLP (ad esempio: Jaeger, Tempo, OpenTelemetry Collector)
OTLP_ENDPOINT=http://localhost:4318
```

**Utilizzo**:
```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

**Vantaggi**:
- Configurazione rapida per lo sviluppo
- Nessun file di configurazione necessario
- Facile da sovrascrivere negli ambienti containerizzati

### Priorità di configurazione

Quando vengono utilizzati entrambi i metodi, il file di configurazione ha la precedenza:

1. Se `CONFIG_FILE` è impostato e contiene una configurazione di tracciamento valida → Utilizzare la configurazione del file
2. Altrimenti, se `OTLP_ENABLED=true` e `OTLP_ENDPOINT` è impostato → Utilizzare le variabili d'ambiente
3. Altrimenti → Il tracciamento è disabilitato

## Span principali

### Span richiesta HTTP

Tutte le richieste HTTP creano automaticamente span con i seguenti attributi:
- `http.method`: Metodo HTTP
- `http.url`: URL della richiesta
- `http.status_code`: Codice di stato della risposta
- `http.user_agent`: User agent
- `http.remote_addr`: Indirizzo del client

### Span query utente (`warden.get_user`)

Le query all'endpoint `/user` creano span dedicati contenenti:
- `warden.query.phone`: Numero di telefono interrogato (mascherado)
- `warden.query.mail`: Email interrogata (mascherada)
- `warden.query.user_id`: ID utente interrogato
- `warden.user.found`: Se l'utente è stato trovato
- `warden.user.id`: ID utente trovato

## Esempi di utilizzo

### Avviare Warden con tracciamento abilitato

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### Utilizzare il tracciamento nel codice

```go
import "github.com/soulteary/warden/internal/tracing"

// Creare uno span figlio
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// Impostare attributi
span.SetAttributes(attribute.String("key", "value"))

// Registrare un errore
if err != nil {
    tracing.RecordError(span, err)
}
```

## Integrazione con Stargate e Herald

Il tracciamento di Warden si integra automaticamente con il contesto di tracciamento dei servizi Stargate e Herald:

1. **Stargate** passa il contesto di traccia tramite header HTTP quando chiama Warden
2. **Warden** estrae automaticamente e continua la catena di tracciamento
3. Gli span di tutti e tre i servizi appaiono nella stessa traccia

## Backend di tracciamento supportati

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **Altri backend compatibili OTLP**

## Considerazioni sulle prestazioni

- Il tracciamento utilizza l'esportazione in batch per impostazione predefinita, minimizzando l'impatto sulle prestazioni
- Il volume dei dati di traccia può essere controllato tramite il tasso di campionamento
- Gli ambienti di produzione dovrebbero utilizzare strategie di campionamento (attualmente campionamento completo, adatto allo sviluppo)

## Risoluzione dei problemi

### Tracciamento non abilitato

Controllare le variabili d'ambiente:
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### I dati di traccia non raggiungono il backend

1. Verificare se l'endpoint OTLP è accessibile
2. Verificare la connessione di rete
3. Controllare i messaggi di errore nei log di Warden

### Span mancanti

Assicurarsi di utilizzare `r.Context()` per passare il contesto nella gestione delle richieste, piuttosto che creare un nuovo contesto.
