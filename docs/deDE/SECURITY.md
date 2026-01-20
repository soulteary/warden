# Sicherheitsdokumentation

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](SECURITY.md) | [한국어](../koKR/SECURITY.md)

Dieses Dokument erläutert die Sicherheitsfunktionen von Warden, die Sicherheitskonfiguration und bewährte Praktiken.


## Implementierte Sicherheitsfunktionen

1. **API-Authentifizierung**: Unterstützt API-Schlüssel-Authentifizierung zum Schutz sensibler Endpunkte
2. **SSRF-Schutz**: Validiert Remote-Konfigurations-URLs streng, um Server-Side Request Forgery-Angriffe zu verhindern
3. **Eingabevalidierung**: Validiert alle Eingabeparameter streng, um Injection-Angriffe zu verhindern
4. **Rate Limiting**: IP-basierte Rate-Limiting zur Verhinderung von DDoS-Angriffen
5. **TLS-Überprüfung**: Produktionsumgebungen erzwingen TLS-Zertifikatsüberprüfung
6. **Fehlerbehandlung**: Produktionsumgebungen verbergen detaillierte Fehlerinformationen, um Informationslecks zu verhindern
7. **Sicherheitsantwort-Header**: Fügt automatisch sicherheitsbezogene HTTP-Antwort-Header hinzu
8. **IP-Whitelist**: Unterstützt die Konfiguration der IP-Whitelist für Health-Check-Endpunkte
9. **Konfigurationsdatei-Validierung**: Verhindert Path-Traversal-Angriffe
10. **JSON-Größenlimits**: Begrenzt die Größe des JSON-Antwortkörpers, um Speichererschöpfungsangriffe zu verhindern

## Sicherheitsbest Practices

### 1. Produktionsumgebungskonfiguration

**Erforderliche Konfiguration**:
- Muss die Umgebungsvariable `API_KEY` setzen
- `MODE=production` setzen, um den Produktionsmodus zu aktivieren
- `TRUSTED_PROXY_IPS` konfigurieren, um die Client-IP korrekt zu erhalten
- `HEALTH_CHECK_IP_WHITELIST` verwenden, um den Zugriff auf Health-Checks einzuschränken

### 2. Verwaltung sensibler Informationen

**Empfohlene Praktiken**:
- ✅ Umgebungsvariablen verwenden, um Passwörter und Schlüssel zu speichern
- ✅ Passwortdateien (`REDIS_PASSWORD_FILE`) verwenden, um Redis-Passwörter zu speichern
- ✅ Platzhalter oder Kommentare in Konfigurationsdateien verwenden
- ✅ Sicherstellen, dass die Berechtigungen der Konfigurationsdateien korrekt gesetzt sind (z. B. `chmod 600`)

### 3. Netzwerksicherheit

**Erforderliche Konfiguration**:
- Produktionsumgebungen müssen HTTPS verwenden
- Firewall-Regeln konfigurieren, um den Zugriff einzuschränken
- Abhängigkeiten regelmäßig aktualisieren, um bekannte Schwachstellen zu beheben

## API-Sicherheit

### API-Schlüssel-Authentifizierung

Einige API-Endpunkte erfordern API-Schlüssel-Authentifizierung.

**Authentifizierungsmethoden**:
1. **X-API-Key Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Rate Limiting

Standardmäßig sind API-Anfragen durch Rate Limiting geschützt:
- **Limit**: 60 Anfragen pro Minute
- **Fenster**: 1 Minute
- **Überschreitung**: Gibt `429 Too Many Requests` zurück

## Schwachstellenmeldung

Wenn Sie eine Sicherheitsschwachstelle entdecken, melden Sie diese bitte über:

1. **GitHub Security Advisory** (Bevorzugt)
   - Gehen Sie zur Registerkarte [Security](https://github.com/soulteary/warden/security) im Repository
   - Klicken Sie auf "Report a vulnerability"
   - Füllen Sie das Security Advisory-Formular aus

2. **E-Mail** (Wenn GitHub Security Advisory nicht verfügbar ist)
   - Senden Sie eine E-Mail an die Projektbetreuer
   - Fügen Sie eine detaillierte Beschreibung der Schwachstelle bei

## Verwandte Dokumentation

- [Konfigurationsdokumentation](CONFIGURATION.md) - Erfahren Sie mehr über sicherheitsbezogene Konfigurationsoptionen
- [Bereitstellungsdokumentation](DEPLOYMENT.md) - Erfahren Sie mehr über Bereitstellungsempfehlungen für Produktionsumgebungen
- [API-Dokumentation](API.md) - Erfahren Sie mehr über API-Sicherheitsfunktionen
