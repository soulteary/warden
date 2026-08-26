# Beitragsleitfaden

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Vielen Dank für Ihr Interesse am Warden-Projekt! Wir begrüßen alle Formen von Beiträgen.


## 📋 Inhaltsverzeichnis

- [Wie man Beiträgt](#wie-man-beiträgt)
- [Entwicklungsumgebung einrichten](#entwicklungsumgebung-einrichten)
- [Code-Standards](#code-standards)
- [Commit-Standards](#commit-standards)
- [Pull Request Prozess](#pull-request-prozess)
- [Fehlerberichte und Funktionsanfragen](#fehlerberichte-und-funktionsanfragen)

## 🚀 Wie man Beiträgt

Sie können auf folgende Weise beitragen:

- **Fehler Melden**: Probleme in GitHub Issues melden
- **Funktionen Vorschlagen**: Neue Funktionsideen in GitHub Issues vorschlagen
- **Code Einreichen**: Code-Verbesserungen über Pull Requests einreichen
- **Dokumentation Verbessern**: Helfen Sie, die Projektdokumentation zu verbessern
- **Fragen Beantworten**: Anderen Benutzern in Issues helfen

Wenn Sie an diesem Projekt teilnehmen, respektieren Sie bitte alle Mitwirkenden, akzeptieren Sie konstruktive Kritik und konzentrieren Sie sich auf das, was für das Projekt am besten ist.

## 🛠️ Entwicklungsumgebung einrichten

### Voraussetzungen

- Go 1.26 oder höher
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

# 5. Lokalen Dienst starten (stellen Sie sicher, dass Redis läuft)
go run .
```

## 📝 Code-Standards

Bitte befolgen Sie diese Code-Standards:

1. **Go Offizielle Code-Standards Befolgen**: [Effective Go](https://go.dev/doc/effective_go)
2. **Code Formatieren**: `go fmt ./...` ausführen
3. **Code Prüfen**: `golangci-lint` oder `go vet ./...` verwenden
4. **Tests Schreiben**: Neue Funktionen müssen Tests enthalten
5. **Kommentare Hinzufügen**: Öffentliche Funktionen und Typen müssen Dokumentationskommentare haben
6. **Konstanten Benennung**: Alle Konstanten müssen den `ALL_CAPS` (UPPER_SNAKE_CASE) Benennungsstil verwenden

Für detaillierte Code-Stil-Richtlinien konsultieren Sie bitte [CODE_STYLE.md](CODE_STYLE.md).

## 📦 Commit-Standards

### Commit-Nachrichtenformat

Wir verwenden den [Conventional Commits](https://www.conventionalcommits.org/) Standard:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Typen

- `feat`: Neue Funktion
- `fix`: Fehlerbehebung
- `docs`: Dokumentationsaktualisierung
- `style`: Code-Formatierungsanpassung (beeinflusst die Code-Ausführung nicht)
- `refactor`: Code-Refaktorierung
- `perf`: Leistungsoptimierung
- `test`: Testbezogen
- `chore`: Änderungen am Build-Prozess oder Hilfswerkzeugen

## 🔄 Pull Request Prozess

### Pull Request Erstellen

```bash
# 1. Funktionsbranch erstellen
git checkout -b feature/your-feature-name

# 2. Änderungen vornehmen und committen
git add .
git commit -m "feat: Neue Funktion hinzufügen"

# 3. Upstream-Code synchronisieren
git fetch upstream
git rebase upstream/main

# 4. Branch pushen und PR erstellen
git push origin feature/your-feature-name
```

### Pull Request Checkliste

Stellen Sie vor dem Einreichen einer Pull Request sicher, dass:

- [ ] Code den Projekt-Code-Standards entspricht
- [ ] Alle Tests bestehen (`go test ./...`)
- [ ] Code formatiert ist (`go fmt ./...`)
- [ ] Notwendige Tests hinzugefügt wurden
- [ ] Verwandte Dokumentation aktualisiert wurde
- [ ] Commit-Nachricht den [Commit-Standards](#commit-standards) entspricht
- [ ] Code Lint-Prüfungen besteht

Alle Pull Requests erfordern eine Code-Überprüfung. Bitte reagieren Sie umgehend auf Überprüfungskommentare.

## 🐛 Fehlerberichte und Funktionsanfragen

Bitte suchen Sie vor dem Erstellen einer Issue in den vorhandenen Issues, um zu bestätigen, dass das Problem oder die Funktion nicht gemeldet wurde.

## 🎯 Erste Schritte

Wenn Sie beitragen möchten, aber nicht wissen, wo Sie anfangen sollen, können Sie sich auf Folgendes konzentrieren:

- Mit `good first issue` markierte Issues
- Mit `help wanted` markierte Issues
- `TODO` Kommentare im Code
- Dokumentationsverbesserungen (Tippfehler korrigieren, Klarheit verbessern, Beispiele hinzufügen)

Wenn Sie Fragen haben, konsultieren Sie bitte vorhandene Issues und Pull Requests oder fragen Sie in relevanten Issues.

---

Vielen Dank nochmals für Ihren Beitrag zum Warden-Projekt! 🎉
