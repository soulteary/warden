# Warden

> 🌐 **Language / 语言**: [English](README.en.md) | [中文](README.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

Ein hochperformanter AllowList-Benutzerdatendienst, der die Datensynchronisation und -zusammenführung aus lokalen und Remote-Konfigurationsquellen unterstützt.

![Warden](.github/assets/banner.jpg)

> **Warden** (Der Wächter) — Der Wächter des Stargate, der entscheidet, wer passieren darf und wer abgelehnt wird. Genau wie der Wächter des Stargate das Stargate bewacht, bewacht Warden Ihre AllowList und stellt sicher, dass nur autorisierte Benutzer passieren können.

## 📋 Projektübersicht

Warden ist ein leichtgewichtiger HTTP-API-Dienst, der in Go entwickelt wurde und hauptsächlich zur Bereitstellung und Verwaltung von AllowList-Benutzerdaten (Telefonnummern und E-Mail-Adressen) verwendet wird. Der Dienst unterstützt das Abrufen von Daten aus lokalen Konfigurationsdateien und Remote-APIs und bietet mehrere Datenzusammenführungsstrategien, um die Echtzeitleistung und Zuverlässigkeit der Daten sicherzustellen.

## ✨ Hauptfunktionen

- 🚀 **Hohe Leistung**: Unterstützt über 5000 Anfragen pro Sekunde mit einer durchschnittlichen Latenz von 21ms
- 🔄 **Mehrere Datenquellen**: Unterstützt sowohl lokale Konfigurationsdateien als auch Remote-APIs
- 🎯 **Flexible Strategien**: Bietet 6 Datenzusammenführungsmodi (Remote-zuerst, lokal-zuerst, nur Remote, nur lokal usw.)
- ⏰ **Geplante Updates**: Geplante Aufgaben basierend auf Redis-Verteilte Sperren für automatische Datensynchronisation
- 📦 **Containerisierte Bereitstellung**: Vollständige Docker-Unterstützung, sofort einsatzbereit
- 📊 **Strukturierte Protokollierung**: Verwendet zerolog, um detaillierte Zugriffs- und Fehlerprotokolle bereitzustellen
- 🔒 **Verteilte Sperren**: Verwendet Redis, um sicherzustellen, dass geplante Aufgaben in verteilten Umgebungen nicht wiederholt ausgeführt werden

## 🏗️ Architekturdesign

Warden verwendet ein geschichtetes Architekturdesign, einschließlich HTTP-Schicht, Geschäftsschicht und Infrastrukturschicht. Das System unterstützt mehrere Datenquellen, mehrstufiges Caching und verteilte Sperrmechanismen.

Für detaillierte Architekturdokumentation siehe: [Architekturdesign-Dokumentation](docs/enUS/ARCHITECTURE.md)

## 📦 Installation und Ausführung

> 💡 **Schnellstart**: Möchten Sie Warden schnell erleben? Schauen Sie sich unsere [Schnellstart-Beispiele](example/README.en.md) an:
> - [Einfaches Beispiel](example/basic/README.en.md) - Grundlegende Verwendung, nur lokale Datendatei
> - [Erweitertes Beispiel](example/advanced/README.en.md) - Vollständige Funktionen, einschließlich Remote-API und Mock-Service

### Voraussetzungen

- Go 1.25+ (siehe [go.mod](go.mod))
- Redis (für verteilte Sperren und Caching)
- Docker (optional, für containerisierte Bereitstellung)

### Schnellstart

1. **Projekt klonen**
```bash
git clone <repository-url>
cd warden
```

2. **Abhängigkeiten installieren**
```bash
go mod download
```

3. **Lokale Datendatei konfigurieren**
Erstellen Sie eine `data.json`-Datei (siehe `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

4. **Service ausführen**
```bash
go run main.go
```

Für detaillierte Konfigurations- und Bereitstellungsanweisungen siehe:
- [Konfigurationsdokumentation](docs/enUS/CONFIGURATION.md) - Erfahren Sie mehr über alle Konfigurationsoptionen
- [Bereitstellungsdokumentation](docs/enUS/DEPLOYMENT.md) - Erfahren Sie mehr über Bereitstellungsmethoden

## ⚙️ Konfiguration

Warden unterstützt mehrere Konfigurationsmethoden: Befehlszeilenargumente, Umgebungsvariablen und Konfigurationsdateien. Das System bietet 6 Datenzusammenführungsmodi mit flexiblen Konfigurationsstrategien.

Für detaillierte Konfigurationsdokumentation siehe: [Konfigurationsdokumentation](docs/enUS/CONFIGURATION.md)

## 📡 API-Dokumentation

Warden bietet eine vollständige RESTful-API mit Unterstützung für Benutzerlistenabfragen, Paginierung, Gesundheitsprüfungen usw. Das Projekt bietet auch OpenAPI 3.0-Spezifikationsdokumentation.

Für detaillierte API-Dokumentation siehe: [API-Dokumentation](docs/enUS/API.md)

OpenAPI-Spezifikationsdatei: [openapi.yaml](openapi.yaml)

## 🔌 SDK-Verwendung

Warden bietet ein Go-SDK zur einfachen Integration in andere Projekte. Das SDK bietet einfache API-Schnittstellen mit Unterstützung für Caching, Authentifizierung usw.

Für detaillierte SDK-Dokumentation siehe: [SDK-Dokumentation](docs/enUS/SDK.md)

## 🐳 Docker-Bereitstellung

Warden unterstützt vollständige Docker- und Docker Compose-Bereitstellung, sofort einsatzbereit.

### Schnellstart mit vorgefertigtem Image (Empfohlen)

Verwenden Sie das vorgefertigte Image von GitHub Container Registry (GHCR), um schnell ohne lokalen Build zu starten:

```bash
# Image der neuesten Version abrufen
docker pull ghcr.io/soulteary/warden:latest

# Container ausführen (Basisbeispiel)
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Tipp**: Die Verwendung vorgefertigter Images ermöglicht es Ihnen, schnell ohne lokale Build-Umgebung zu starten. Images werden automatisch aktualisiert, um sicherzustellen, dass Sie die neueste Version verwenden.

### Verwendung von Docker Compose

> 🚀 **Schnelle Bereitstellung**: Schauen Sie sich das [Beispielverzeichnis](example/README.en.md) für vollständige Docker Compose-Konfigurationsbeispiele an

Für detaillierte Bereitstellungsdokumentation siehe: [Bereitstellungsdokumentation](docs/enUS/DEPLOYMENT.md)

## 📊 Leistungsmetriken

Basierend auf wrk-Lasttest-Ergebnissen (30-Sekunden-Test, 16 Threads, 100 Verbindungen):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Durchschnittliche Latenz: 21.30ms
Maximale Latenz: 226.09ms
```

## 📁 Projektstruktur

```
warden/
├── main.go                 # Programmeinstiegspunkt
├── data.example.json      # Beispiel für lokale Datendatei
├── config.example.yaml    # Beispiel für Konfigurationsdatei
├── openapi.yaml           # OpenAPI-Spezifikationsdatei
├── go.mod                 # Go-Moduldefinition
├── docker-compose.yml     # Docker Compose-Konfiguration
├── LICENSE                # Lizenzdatei
├── README.*.md            # Mehrsprachige Projektdokumente (Chinesisch/Englisch/Französisch/Italienisch/Japanisch/Deutsch/Koreanisch)
├── CONTRIBUTING.*.md      # Mehrsprachige Beitragsleitfäden
├── docker/
│   └── Dockerfile         # Docker-Image-Build-Datei
├── docs/                  # Dokumentationsverzeichnis (mehrsprachig)
│   ├── enUS/              # Englische Dokumentation
│   └── zhCN/              # Chinesische Dokumentation
├── example/               # Schnellstart-Beispiele
│   ├── basic/             # Einfaches Beispiel (nur lokale Datei)
│   └── advanced/          # Erweitertes Beispiel (vollständige Funktionen, enthält Mock API)
├── internal/
│   ├── cache/             # Redis-Cache- und Sperr-Implementierung
│   ├── cmd/               # Befehlszeilenargument-Parsing
│   ├── config/            # Konfigurationsverwaltung
│   ├── define/            # Konstantendefinitionen und Datenstrukturen
│   ├── di/                # Abhängigkeitsinjektion
│   ├── errors/            # Fehlerbehandlung
│   ├── logger/            # Protokollierungsinitialisierung
│   ├── metrics/           # Metrikensammlung
│   ├── middleware/        # HTTP-Middleware
│   ├── parser/            # Datenparser (lokal/remote)
│   ├── router/            # HTTP-Routenverarbeitung
│   ├── validator/         # Validator
│   └── version/           # Versionsinformationen
├── pkg/
│   ├── gocron/            # Geplante Aufgabenplaner
│   └── warden/            # Warden SDK
├── scripts/               # Skriptverzeichnis
└── .github/               # GitHub-Konfiguration (CI/CD, Issue/PR-Vorlagen, etc.)
```

## 🔒 Sicherheitsfunktionen

Warden implementiert mehrere Sicherheitsfunktionen, einschließlich API-Authentifizierung, SSRF-Schutz, Ratenbegrenzung, TLS-Überprüfung usw.

Für detaillierte Sicherheitsdokumentation siehe: [Sicherheitsdokumentation](docs/enUS/SECURITY.md)

## 🔧 Entwicklungsleitfaden

> 📚 **Referenzbeispiele**: Schauen Sie sich das [Beispielverzeichnis](example/README.en.md) für vollständige Beispielcode und Konfigurationen für verschiedene Verwendungsszenarien an.

Für detaillierte Entwicklungsdokumentation siehe: [Entwicklungsdokumentation](docs/enUS/DEVELOPMENT.md)

### Codestandards

Das Projekt folgt den offiziellen Go-Codestandards und Best Practices. Für detaillierte Standards siehe:

- [CODE_STYLE.md](docs/enUS/CODE_STYLE.md) - Codestil-Leitfaden
- [CONTRIBUTING.en.md](CONTRIBUTING.en.md) - Beitragsleitfaden

## 📄 Lizenz

Siehe die [LICENSE](LICENSE)-Datei für Details.

## 🤝 Beitragen

Issues und Pull Requests sind willkommen!

## 📞 Kontakt

Bei Fragen oder Vorschlägen kontaktieren Sie uns bitte über Issues.

---

**Version**: Das Programm zeigt Version, Build-Zeit und Code-Version beim Start an (über `warden --version` oder Startprotokolle)
