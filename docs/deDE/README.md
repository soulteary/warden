# Dokumentationsindex

Willkommen zur Warden AllowList Benutzerdatendienst-Dokumentation.

## 🌐 Mehrsprachige Dokumentation

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](README.md) | [한국어](../koKR/README.md)

## 📚 Dokumentenliste

### Kerndokumente

- **[README.md](../../README.deDE.md)** - Projektübersicht und Schnellstart-Anleitung
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technische Architektur und Designentscheidungen

### Detaillierte Dokumente

- **[API.md](API.md)** - Vollständige API-Endpunkt-Dokumentation
  - Benutzerlisten-Abfrage-Endpunkte
  - Paginierungsfunktionalität
  - Health-Check-Endpunkte
  - Fehlerantwortformate

- **[CONFIGURATION.md](CONFIGURATION.md)** - Konfigurationsreferenz
  - Konfigurationsmethoden
  - Erforderliche Konfigurationselemente
  - Optionale Konfigurationselemente
  - Datenzusammenführungsstrategien
  - Konfigurationsbeispiele
  - Konfigurationsbest Practices

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Bereitstellungsanleitung
  - Docker-Bereitstellung (einschließlich GHCR-Images)
  - Docker Compose-Bereitstellung
  - Lokale Bereitstellung
  - Produktionsumgebungs-Bereitstellung
  - Kubernetes-Bereitstellung
  - Leistungsoptimierung

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Entwicklungsanleitung
  - Entwicklungsumgebung einrichten
  - Code-Struktur-Erklärung
  - Testanleitung
  - Beitragsanleitung

- **[SDK.md](SDK.md)** - SDK-Verwendungsdokumentation
  - Go SDK-Installation und Verwendung
  - API-Schnittstellenbeschreibung
  - Beispielcode

- **[SECURITY.md](SECURITY.md)** - Sicherheitsdokumentation
  - Sicherheitsfunktionen
  - Sicherheitskonfiguration
  - Best Practices

- **[CODE_STYLE.md](CODE_STYLE.md)** - Code-Stil-Anleitung
  - Code-Standards
  - Benennungskonventionen
  - Best Practices

## 🌐 Mehrsprachige Unterstützung

Warden unterstützt vollständige Internationalisierungs- (i18N) Funktionalität. Alle API-Antworten, Fehlermeldungen und Protokolle unterstützen Internationalisierung.

### Unterstützte Sprachen

- 🇺🇸 Englisch (en) - Standardsprache
- 🇨🇳 Chinesisch (zh)
- 🇫🇷 Französisch (fr)
- 🇮🇹 Italienisch (it)
- 🇯🇵 Japanisch (ja)
- 🇩🇪 Deutsch (de)
- 🇰🇷 Koreanisch (ko)

### Spracherkennung

Warden unterstützt zwei Spracherkennungsmethoden mit folgender Priorität:

1. **Abfrageparameter**: Sprache über URL-Abfrageparameter `?lang=de` angeben
2. **Accept-Language-Header**: Automatische Erkennung der Browser- oder Client-Spracheinstellung
3. **Standardsprache**: Englisch, wenn nicht angegeben

### Verwendungsbeispiele

#### Sprache über Abfrageparameter angeben

```bash
# Deutsch verwenden
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=de"

# Japanisch verwenden
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"

# Französisch verwenden
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"
```

#### Automatische Erkennung über Accept-Language-Header

```bash
# Browser sendet automatisch Accept-Language-Header
curl -H "X-API-Key: your-key" \
     -H "Accept-Language: de-DE,de;q=0.9,en;q=0.8" \
     "http://localhost:8081/"
```

### Internationalisierungsbereich

Die folgenden Inhalte unterstützen mehrere Sprachen:

- ✅ API-Fehlerantwortmeldungen
- ✅ HTTP-Statuscode-Fehlermeldungen
- ✅ Protokollmeldungen (basierend auf Anforderungskontext)
- ✅ Konfigurations- und Warnmeldungen

### Technische Implementierung

- Verwendet Anforderungskontext zum Speichern von Sprachinformationen, vermeidet globalen Zustand
- Unterstützt threadsichere Sprachumschaltung
- Automatisches Fallback auf Englisch (wenn Übersetzung nicht gefunden)
- Alle Übersetzungen sind in den Code eingebaut, keine externen Dateien erforderlich

### Entwicklungsnotizen

Um neue Übersetzungen hinzuzufügen oder vorhandene Übersetzungen zu ändern, bearbeiten Sie bitte die `translations`-Map in der Datei `internal/i18n/i18n.go`.

## 🚀 Schnellnavigation

### Erste Schritte

1. Lesen Sie [README.deDE.md](../../README.deDE.md), um das Projekt zu verstehen
2. Überprüfen Sie den Abschnitt [Schnellstart](../../README.deDE.md#schnellstart)
3. Beziehen Sie sich auf [Konfiguration](../../README.deDE.md#konfiguration), um den Dienst zu konfigurieren

### Entwickler

1. Lesen Sie [ARCHITECTURE.md](ARCHITECTURE.md), um die Architektur zu verstehen
2. Überprüfen Sie [API.md](API.md), um die API-Schnittstellen zu verstehen
3. Beziehen Sie sich auf die [Entwicklungsanleitung](../../README.deDE.md#entwicklungsanleitung) für die Entwicklung

### Betrieb

1. Lesen Sie [DEPLOYMENT.md](DEPLOYMENT.md), um Bereitstellungsmethoden zu verstehen
2. Überprüfen Sie [CONFIGURATION.md](CONFIGURATION.md), um Konfigurationsoptionen zu verstehen
3. Beziehen Sie sich auf [Leistungsoptimierung](DEPLOYMENT.md#leistungsoptimierung), um den Dienst zu optimieren

## 📖 Dokumentstruktur

```
warden/
├── README.md              # Hauptprojektdokument (Deutsch)
├── README.deDE.md         # Hauptprojektdokument (Deutsch)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Dokumentationsindex (Englisch)
│   │   ├── ARCHITECTURE.md # Architekturdokument (Englisch)
│   │   ├── API.md          # API-Dokument (Englisch)
│   │   ├── CONFIGURATION.md # Konfigurationsreferenz (Englisch)
│   │   ├── DEPLOYMENT.md   # Bereitstellungsanleitung (Englisch)
│   │   ├── DEVELOPMENT.md  # Entwicklungsanleitung (Englisch)
│   │   ├── SDK.md          # SDK-Dokument (Englisch)
│   │   ├── SECURITY.md     # Sicherheitsdokument (Englisch)
│   │   └── CODE_STYLE.md   # Code-Stil (Englisch)
│   └── deDE/
│       ├── README.md       # Dokumentationsindex (Deutsch, diese Datei)
│       ├── ARCHITECTURE.md # Architekturdokument (Deutsch)
│       ├── API.md          # API-Dokument (Deutsch)
│       ├── CONFIGURATION.md # Konfigurationsreferenz (Deutsch)
│       ├── DEPLOYMENT.md   # Bereitstellungsanleitung (Deutsch)
│       ├── DEVELOPMENT.md  # Entwicklungsanleitung (Deutsch)
│       ├── SDK.md          # SDK-Dokument (Deutsch)
│       ├── SECURITY.md     # Sicherheitsdokument (Deutsch)
│       └── CODE_STYLE.md   # Code-Stil (Deutsch)
└── ...
```

## 🔍 Nach Thema finden

### Konfigurationsbezogen

- Umgebungsvariablen-Konfiguration: [CONFIGURATION.md](CONFIGURATION.md)
- Datenzusammenführungsstrategien: [CONFIGURATION.md](CONFIGURATION.md)
- Konfigurationsbeispiele: [CONFIGURATION.md](CONFIGURATION.md)

### API-bezogen

- API-Endpunktliste: [API.md](API.md)
- Fehlerbehandlung: [API.md](API.md)
- Paginierungsfunktionalität: [API.md](API.md)

### Bereitstellungsbezogen

- Docker-Bereitstellung: [DEPLOYMENT.md#docker-bereitstellung](DEPLOYMENT.md#docker-bereitstellung)
- GHCR-Images: [DEPLOYMENT.md#verwenden-von-vorgefertigten-images-empfohlen](DEPLOYMENT.md#verwenden-von-vorgefertigten-images-empfohlen)
- Produktionsumgebung: [DEPLOYMENT.md#produktionsumgebungs-bereitstellungsempfehlungen](DEPLOYMENT.md#produktionsumgebungs-bereitstellungsempfehlungen)
- Kubernetes: [DEPLOYMENT.md#kubernetes-bereitstellung](DEPLOYMENT.md#kubernetes-bereitstellung)

### Architekturbezogen

- Technologie-Stack: [ARCHITECTURE.md](ARCHITECTURE.md)
- Projektstruktur: [ARCHITECTURE.md](ARCHITECTURE.md)
- Kernkomponenten: [ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 Verwendungsempfehlungen

1. **Erstmalige Benutzer**: Beginnen Sie mit [README.deDE.md](../../README.deDE.md) und folgen Sie der Schnellstart-Anleitung
2. **Dienst konfigurieren**: Beziehen Sie sich auf [CONFIGURATION.md](CONFIGURATION.md), um alle Konfigurationsoptionen zu verstehen
3. **Dienst bereitstellen**: Überprüfen Sie [DEPLOYMENT.md](DEPLOYMENT.md), um Bereitstellungsmethoden zu verstehen
4. **Erweiterungen entwickeln**: Lesen Sie [ARCHITECTURE.md](ARCHITECTURE.md), um das Architekturdesign zu verstehen
5. **SDK integrieren**: Beziehen Sie sich auf [SDK.md](SDK.md), um zu erfahren, wie das SDK verwendet wird

## 📝 Dokumentaktualisierungen

Die Dokumentation wird kontinuierlich aktualisiert, während sich das Projekt entwickelt. Wenn Sie Fehler finden oder Ergänzungen benötigen, reichen Sie bitte ein Issue oder Pull Request ein.

## 🤝 Beitragen

Verbesserungen der Dokumentation sind willkommen:

1. Fehler oder Bereiche finden, die verbessert werden müssen
2. Ein Issue einreichen, das das Problem beschreibt
3. Oder direkt einen Pull Request einreichen
