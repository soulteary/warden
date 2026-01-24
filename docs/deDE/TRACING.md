# Warden OpenTelemetry Tracing

Der Warden-Dienst unterstützt OpenTelemetry Distributed Tracing zur Überwachung und Fehlerbehebung von Aufrufketten zwischen Diensten.

## Funktionen

- **Automatische HTTP-Anfrageverfolgung**: Erstellt automatisch Spans für alle HTTP-Anfragen
- **Benutzerabfrageverfolgung**: Fügt detaillierte Verfolgungsinformationen für den `/user`-Endpunkt hinzu
- **Kontextpropagierung**: Unterstützt den W3C Trace Context-Standard und integriert sich nahtlos mit Stargate- und Herald-Diensten
- **Konfigurierbar**: Aktivieren/Deaktivieren über Umgebungsvariablen oder Konfigurationsdateien

## Konfiguration

Warden unterstützt zwei Methoden zur Konfiguration der OpenTelemetry-Verfolgung:

1. **Konfigurationsdatei** (empfohlen für Produktion)
2. **Umgebungsvariablen** (bequem für Entwicklung)

**Priorität**: Die Konfigurationsdatei hat Vorrang vor Umgebungsvariablen.

### Methode 1: Konfigurationsdatei (YAML)

Erstellen Sie eine Konfigurationsdatei (z. B. `config.yaml`) und geben Sie sie über die Umgebungsvariable `CONFIG_FILE` an:

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

**Verwendung**:
```bash
export CONFIG_FILE=/path/to/config.yaml
./warden
```

**Vorteile**:
- Zentrale Konfigurationsverwaltung
- Besser für Produktionsumgebungen
- Unterstützt alle Konfigurationsoptionen in einer Datei

### Methode 2: Umgebungsvariablen

```bash
# OpenTelemetry-Verfolgung aktivieren
OTLP_ENABLED=true

# OTLP-Endpunkt (z. B. Jaeger, Tempo, OpenTelemetry Collector)
OTLP_ENDPOINT=http://localhost:4318
```

**Verwendung**:
```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

**Vorteile**:
- Schnelle Einrichtung für Entwicklung
- Keine Konfigurationsdatei erforderlich
- Einfach in containerisierten Umgebungen zu überschreiben

### Konfigurationspriorität

Wenn beide Methoden verwendet werden, hat die Konfigurationsdatei Vorrang:

1. Wenn `CONFIG_FILE` gesetzt ist und gültige Verfolgungskonfiguration enthält → Verwenden Sie Dateikonfiguration
2. Andernfalls, wenn `OTLP_ENABLED=true` und `OTLP_ENDPOINT` gesetzt ist → Verwenden Sie Umgebungsvariablen
3. Andernfalls → Verfolgung ist deaktiviert

## Kern-Spans

### HTTP-Anfrage-Span

Alle HTTP-Anfragen erstellen automatisch Spans mit den folgenden Attributen:
- `http.method`: HTTP-Methode
- `http.url`: Anfrage-URL
- `http.status_code`: Antwortstatuscode
- `http.user_agent`: User-Agent
- `http.remote_addr`: Client-Adresse

### Benutzerabfrage-Span (`warden.get_user`)

Abfragen an den `/user`-Endpunkt erstellen dedizierte Spans mit:
- `warden.query.phone`: Abgefragte Telefonnummer (maskiert)
- `warden.query.mail`: Abgefragte E-Mail (maskiert)
- `warden.query.user_id`: Abgefragte Benutzer-ID
- `warden.user.found`: Ob Benutzer gefunden wurde
- `warden.user.id`: Gefundene Benutzer-ID

## Verwendungsbeispiele

### Warden mit aktivierter Verfolgung starten

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### Verfolgung im Code verwenden

```go
import "github.com/soulteary/warden/internal/tracing"

// Untergeordneten Span erstellen
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// Attribute setzen
span.SetAttributes(attribute.String("key", "value"))

// Fehler aufzeichnen
if err != nil {
    tracing.RecordError(span, err)
}
```

## Integration mit Stargate und Herald

Die Verfolgung von Warden integriert sich automatisch mit dem Verfolgungskontext der Stargate- und Herald-Dienste:

1. **Stargate** übergibt den Trace-Kontext über HTTP-Header beim Aufruf von Warden
2. **Warden** extrahiert automatisch und setzt die Verfolgungskette fort
3. Spans aller drei Dienste erscheinen in derselben Trace

## Unterstützte Verfolgungs-Backends

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **Andere OTLP-kompatible Backends**

## Leistungsüberlegungen

- Die Verfolgung verwendet standardmäßig Batch-Export, minimiert die Leistungsauswirkungen
- Das Verfolgungsdatenvolumen kann über die Abtastrate gesteuert werden
- Produktionsumgebungen sollten Abtaststrategien verwenden (derzeit vollständige Abtastung, geeignet für Entwicklung)

## Fehlerbehebung

### Verfolgung nicht aktiviert

Umgebungsvariablen überprüfen:
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### Verfolgungsdaten erreichen Backend nicht

1. Überprüfen, ob der OTLP-Endpunkt erreichbar ist
2. Netzwerkverbindung überprüfen
3. Fehlermeldungen in Warden-Protokollen überprüfen

### Fehlende Spans

Stellen Sie sicher, dass Sie `r.Context()` verwenden, um den Kontext in der Anfrageverarbeitung zu übergeben, anstatt einen neuen Kontext zu erstellen.
