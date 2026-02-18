# Warden

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/warden/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/warden)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/warden)](https://goreportcard.com/report/github.com/soulteary/warden)

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

Ein hochperformanter AllowList-Benutzerdatendienst, der die Datensynchronisation und -zusammenführung aus lokalen und Remote-Konfigurationsquellen unterstützt.

![Warden](.github/assets/banner.jpg)

> **Warden** (Der Wächter) — Der Wächter des Stargate, der entscheidet, wer passieren darf und wer abgelehnt wird. Genau wie der Wächter des Stargate das Stargate bewacht, bewacht Warden Ihre AllowList und stellt sicher, dass nur autorisierte Benutzer passieren können.

## 📋 Übersicht

Warden ist ein leichtgewichtiger HTTP-API-Dienst, der in Go entwickelt wurde und hauptsächlich zur Bereitstellung und Verwaltung von AllowList-Benutzerdaten (Telefonnummern und E-Mail-Adressen) verwendet wird. Der Dienst unterstützt das Abrufen von Daten aus lokalen Konfigurationsdateien und Remote-APIs und bietet mehrere Datenzusammenführungsstrategien, um die Echtzeitleistung und Zuverlässigkeit der Daten sicherzustellen.

Warden kann **eigenständig** verwendet werden oder mit anderen Diensten (wie Stargate und Herald) als Teil einer größeren Authentifizierungsarchitektur integriert werden. Detaillierte Architekturinformationen finden Sie in der [Architekturdokumentation](docs/enUS/ARCHITECTURE.md).

## ✨ Hauptfunktionen

- 🚀 **Hohe Leistung**: Über 5000 Anfragen pro Sekunde mit einer durchschnittlichen Latenz von 21ms
- 🔄 **Mehrere Datenquellen**: Lokale Konfigurationsdateien und Remote-APIs
- 🎯 **Flexible Strategien**: 6 Datenzusammenführungsmodi (Remote-zuerst, lokal-zuerst, nur Remote, nur lokal usw.)
- ⏰ **Geplante Updates**: Automatische Datensynchronisation mit Redis-Verteilte Sperren
- 📦 **Containerisierte Bereitstellung**: Vollständige Docker-Unterstützung, sofort einsatzbereit
- 🌐 **Mehrsprachige Unterstützung**: 7 Sprachen mit automatischer Spracherkennung

## 🚀 Schnellstart

### Option 1: Docker (Empfohlen)

Der schnellste Weg zum Einstieg ist die Verwendung des vorgefertigten Docker-Images:

```bash
# Neuestes Image abrufen
docker pull ghcr.io/soulteary/warden:latest

# Datendatei erstellen
cat > data.json <<EOF
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
EOF

# Container ausführen
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Tipp**: Vollständige Beispiele mit Docker Compose finden Sie im [Beispielverzeichnis](example/README.md).

### Option 2: Aus dem Quellcode

1. **Projekt klonen und erstellen**
```bash
git clone <repository-url>
cd warden
go mod download
```

2. **Datendatei erstellen**
Erstellen Sie eine `data.json`-Datei (siehe `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

3. **Service ausführen**
```bash
go run . --api-key your-api-key-here
```

## ⚙️ Wesentliche Konfiguration

Warden unterstützt die Konfiguration über Befehlszeilenargumente, Umgebungsvariablen und Konfigurationsdateien. Die folgenden sind die wichtigsten Einstellungen:

| Einstellung | Umgebungsvariable | Beschreibung | Erforderlich |
|-------------|-------------------|--------------|--------------|
| Port | `PORT` | HTTP-Server-Port (Standard: 8081) | Nein |
| API-Schlüssel | `API_KEY` | API-Authentifizierungsschlüssel (für Produktion empfohlen) | Empfohlen |
| Redis | `REDIS` | Redis-Adresse für Caching und verteilte Sperren (z.B. `localhost:6379`) | Optional |
| Datendatei | - | Pfad zur lokalen Datendatei (Standard: `data.json`) | Ja* |
| Remote-Konfiguration | `CONFIG` | Remote-API-URL zum Abrufen von Daten | Optional |

\* Erforderlich, wenn keine Remote-API verwendet wird

Vollständige Konfigurationsoptionen finden Sie in der [Konfigurationsdokumentation](docs/enUS/CONFIGURATION.md).

## 📡 API-Verwendung

Warden bietet eine RESTful-API zum Abfragen von Benutzerlisten, Paginierung und Gesundheitsprüfungen. Der Dienst unterstützt mehrsprachige Antworten über den Abfrageparameter `?lang=xx` oder den `Accept-Language`-Header.

**Beispiel**:
```bash
# Benutzer abfragen
curl -H "X-API-Key: your-key" "http://localhost:8081/"

# Gesundheitsprüfung
curl "http://localhost:8081/health"
```

Vollständige API-Dokumentation finden Sie in der [API-Dokumentation](docs/enUS/API.md) oder der [OpenAPI-Spezifikation](openapi.yaml).

## 📊 Leistung

Basierend auf wrk-Stresstest (30s, 16 Threads, 100 Verbindungen):
- **Anfragen/Sekunde**: 5038.81
- **Durchschnittliche Latenz**: 21.30ms
- **Maximale Latenz**: 226.09ms

## 📚 Dokumentation

### Kern-Dokumentation

- **[Architektur](docs/enUS/ARCHITECTURE.md)** - Technische Architektur und Designentscheidungen
- **[API-Referenz](docs/enUS/API.md)** - Vollständige API-Endpunkt-Dokumentation
- **[Konfiguration](docs/enUS/CONFIGURATION.md)** - Konfigurationsreferenz und Beispiele
- **[Bereitstellung](docs/enUS/DEPLOYMENT.md)** - Bereitstellungsanleitung (Docker, Kubernetes usw.)

### Zusätzliche Ressourcen

- **[Entwicklungsleitfaden](docs/enUS/DEVELOPMENT.md)** - Entwicklungsumgebung einrichten und Beitragsleitfaden
- **[Sicherheit](docs/enUS/SECURITY.md)** - Sicherheitsfunktionen und Best Practices
- **[SDK](docs/enUS/SDK.md)** - Go SDK-Verwendungsdokumentation
- **[Beispiele](example/README.md)** - Schnellstart-Beispiele (grundlegend und erweitert)

## 📄 Lizenz

Siehe die [LICENSE](LICENSE)-Datei für Details.

## 🤝 Beitragen

Willkommen zur Einreichung von Issues und Pull Requests! Siehe [CONTRIBUTING.md](docs/enUS/CONTRIBUTING.md) für Richtlinien.
