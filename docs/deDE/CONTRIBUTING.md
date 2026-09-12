# Beitragsleitfaden

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Vielen Dank für Ihr Interesse am Warden-Projekt! Wir freuen uns über Beiträge jeder Art.

## 📋 Inhaltsverzeichnis

- [Wie Sie beitragen können](#wie-sie-beitragen-können)
- [Einrichtung der Entwicklungsumgebung](#einrichtung-der-entwicklungsumgebung)
- [Code-Standards](#code-standards)
- [Übersetzungsrichtlinie](#übersetzungsrichtlinie)
- [Commit-Standards](#commit-standards)
- [Pull-Request-Prozess](#pull-request-prozess)
- [Fehlerberichte und Feature-Wünsche](#fehlerberichte-und-feature-wünsche)

## 🚀 Wie Sie beitragen können

Sie können auf folgende Arten beitragen:

- **Fehler melden**: Probleme in GitHub Issues melden
- **Features vorschlagen**: Ideen für neue Funktionen in GitHub Issues einbringen
- **Code einreichen**: Verbesserungen am Code per Pull Request einreichen
- **Dokumentation verbessern**: Helfen Sie mit, die Projektdokumentation zu verbessern
- **Fragen beantworten**: Unterstützen Sie andere Nutzerinnen und Nutzer in den Issues

Bitte begegnen Sie allen Mitwirkenden respektvoll, nehmen Sie konstruktive Kritik an und behalten Sie im Blick, was für das Projekt am besten ist.

## 🛠️ Einrichtung der Entwicklungsumgebung

### Voraussetzungen

- Go 1.27 oder höher
- Redis (für Tests)
- Git

### Schnellstart

```bash
# 1. Projekt forken und klonen
git clone https://github.com/your-username/warden.git
cd warden

# 2. Upstream-Repository hinzufügen
git remote add upstream https://github.com/soulteary/warden.git

# 3. Abhängigkeiten installieren
go mod download

# 4. Tests ausführen
go test ./...

# 5. Dienst lokal starten (Redis muss laufen)
go run .
```

## 📝 Code-Standards

Bitte halten Sie sich an folgende Code-Standards:

1. **Offiziellen Go-Standards folgen**: [Effective Go](https://go.dev/doc/effective_go)
2. **Code formatieren**: `go fmt ./...` ausführen
3. **Code prüfen**: `golangci-lint` oder `go vet ./...` verwenden
4. **Tests schreiben**: Neue Funktionen müssen Tests enthalten
5. **Kommentare ergänzen**: Öffentliche Funktionen und Typen benötigen Dokumentationskommentare
6. **Benennung von Konstanten**: Alle Konstanten müssen den Stil `ALL_CAPS` (UPPER_SNAKE_CASE) verwenden

Ausführliche Richtlinien zum Code-Stil finden Sie in [CODE_STYLE.md](CODE_STYLE.md).

## 🌐 Übersetzungsrichtlinie

Warden liefert Dokumentation und Laufzeitmeldungen in sieben Sprachen aus. Sie werden **nicht**
alle nach demselben Maßstab gepflegt, und so zu tun, als wäre es anders, ist genau der Grund,
weshalb fünf Locales unbemerkt 18 Übersetzungsschlüssel und mehrere Dokumente hinterherhinkten.

**Stufen**

| Stufe | Sprachen | Erwartung |
| --- | --- | --- |
| Maßgeblich | Englisch (`enUS`), vereinfachtes Chinesisch (`zhCN`) | Werden im selben Pull Request wie die Änderung aktualisiert. Ein PR, der Verhalten ändert, ohne beide zu aktualisieren, ist unvollständig. |
| Nach bestem Bemühen | `deDE`, `frFR`, `itIT`, `jaJP`, `koKR` | Dürfen hinterherhinken. Jedes zurückliegende Dokument trägt ein Banner, das auf die maßgeblichen Fassungen verweist. |

**Laufzeit-Strings sind nicht „nach bestem Bemühen“.** `locales/*.json` wird durch
`go test ./locales/` erzwungen; der Test schlägt fehl, sobald ein Locale:

- einen Schlüssel fehlen lässt, den `en.json` definiert (oder einen definiert, den es dort nicht gibt),
- eine andere Abfolge von `printf`-Verben (`%s`, `%d`) als die englische Quelle aufweist oder
- einen Wert enthält, der byteweise mit dem englischen identisch ist (eine unübersetzte Zeichenkette).

Fehlende Schlüssel fallen zur Laufzeit auf Englisch zurück, sodass nichts sichtbar kaputtgeht —
und genau deshalb gibt es diese Prüfung. Wenn Sie eine benutzersichtbare Meldung hinzufügen,
ergänzen Sie den Schlüssel im selben Commit in **allen sieben** Locale-Dateien. Ist ein Wert
berechtigterweise mit dem englischen identisch (ein Lehnwort, ein Protokollname), tragen Sie
ihn mit einem Kommentar in `intentionallyIdentical` in `locales/locales_test.go` ein, statt
die Prüfung zu entfernen.

Führen Sie `make docs-parity` aus, um zu sehen, wie weit jedes übersetzte Dokument von `enUS` abgewichen ist.

## 📦 Commit-Standards

### Format der Commit-Nachricht

Wir verwenden den Standard [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Typen

- `feat`: Neue Funktion
- `fix`: Fehlerbehebung
- `docs`: Aktualisierung der Dokumentation
- `style`: Anpassung der Code-Formatierung (ohne Auswirkung auf die Ausführung)
- `refactor`: Refaktorierung des Codes
- `perf`: Leistungsoptimierung
- `test`: Tests betreffend
- `chore`: Änderungen am Build-Prozess oder an Hilfswerkzeugen

### Beispiele

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

## 🔄 Pull-Request-Prozess

### Pull Request erstellen

```bash
# 1. Feature-Branch erstellen
git checkout -b feature/your-feature-name

# 2. Änderungen vornehmen und committen
git add .
git commit -m "feat: Add new feature"

# 3. Upstream-Code synchronisieren
git fetch upstream
git rebase upstream/main

# 4. Branch pushen und PR erstellen
git push origin feature/your-feature-name
```

### Checkliste für Pull Requests

Stellen Sie vor dem Einreichen eines Pull Requests sicher:

- [ ] Der Code folgt den Code-Standards des Projekts
- [ ] Alle Tests laufen durch (`go test ./...`)
- [ ] Der Code ist formatiert (`go fmt ./...`)
- [ ] Notwendige Tests wurden ergänzt
- [ ] Zugehörige Dokumentation wurde aktualisiert
- [ ] Die Commit-Nachricht folgt den [Commit-Standards](#commit-standards)
- [ ] Der Code besteht die Lint-Prüfungen

Alle Pull Requests durchlaufen ein Code-Review. Bitte reagieren Sie zeitnah auf Review-Kommentare.

## 🐛 Fehlerberichte und Feature-Wünsche

Bitte durchsuchen Sie vor dem Anlegen eines Issues die bestehenden Issues, um sicherzustellen, dass das Problem oder die Funktion noch nicht gemeldet wurde.

### Vorlage für Fehlerberichte

```markdown
**Beschreibung**
Beschreiben Sie den Fehler klar und knapp.

**Schritte zur Reproduktion**
1. '...' ausführen
2. Fehler beobachten

**Erwartetes Verhalten**
Beschreiben Sie klar und knapp, was Sie erwartet haben.

**Tatsächliches Verhalten**
Beschreiben Sie klar und knapp, was tatsächlich passiert ist.

**Umgebungsinformationen**
- Betriebssystem: [z. B. macOS 12.0]
- Go-Version: [z. B. 1.27]
- Redis-Version: [z. B. 7.0]
```

### Vorlage für Feature-Wünsche

```markdown
**Beschreibung der Funktion**
Beschreiben Sie die gewünschte Funktion klar und knapp.

**Problembeschreibung**
Welches Problem löst diese Funktion? Warum wird sie benötigt?

**Lösungsvorschlag**
Beschreiben Sie klar und knapp, wie Sie sich die Umsetzung vorstellen.
```

## 🎯 Erste Schritte

Wenn Sie beitragen möchten, aber nicht wissen, wo Sie anfangen sollen, empfiehlt sich ein Blick auf:

- Issues mit der Kennzeichnung `good first issue`
- Issues mit der Kennzeichnung `help wanted`
- `TODO`-Kommentare im Code
- Verbesserungen an der Dokumentation (Tippfehler beheben, Verständlichkeit erhöhen, Beispiele ergänzen)

Bei Fragen sehen Sie sich bitte bestehende Issues und Pull Requests an oder fragen Sie im passenden Issue nach.

---

Nochmals vielen Dank für Ihren Beitrag zum Warden-Projekt! 🎉
