# Warden

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/warden/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/warden)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/warden)](https://goreportcard.com/report/github.com/soulteary/warden)

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

Un service de données utilisateur de liste d'autorisation (AllowList) haute performance qui prend en charge la synchronisation et la fusion de données à partir de sources de configuration locales et distantes.

![Warden](.github/assets/banner.jpg)

> **Warden** (Le Gardien) — Le gardien de la Porte des Étoiles qui décide qui peut passer et qui sera refusé. Tout comme le Gardien de Stargate garde la Porte des Étoiles, Warden garde votre liste d'autorisation, garantissant que seuls les utilisateurs autorisés peuvent passer.

## 📋 Aperçu

Warden est un service API HTTP léger développé en Go, principalement utilisé pour fournir et gérer les données utilisateur de liste d'autorisation (numéros de téléphone et adresses e-mail). Le service prend en charge la récupération de données à partir de fichiers de configuration locaux et d'API distantes, et fournit plusieurs stratégies de fusion de données pour assurer la performance et la fiabilité des données en temps réel.

Warden peut être utilisé **de manière autonome** ou intégré avec d'autres services (tels que Stargate et Herald) dans le cadre d'une architecture d'authentification plus large. Pour des informations détaillées sur l'architecture, consultez la [Documentation de l'Architecture](docs/enUS/ARCHITECTURE.md).

## ✨ Fonctionnalités Principales

- 🚀 **Haute Performance**: Plus de 5000 requêtes par seconde avec une latence moyenne de 21ms
- 🔄 **Sources de Données Multiples**: Fichiers de configuration locaux et API distantes
- 🎯 **Stratégies Flexibles**: 6 modes de fusion de données (priorité distante, priorité locale, distant uniquement, local uniquement, etc.)
- ⏰ **Mises à Jour Planifiées**: Synchronisation automatique des données avec verrous distribués Redis
- 📦 **Déploiement Conteneurisé**: Support Docker complet, prêt à l'emploi
- 🌐 **Support Multi-langues**: 7 langues avec détection automatique de la langue

## 🚀 Démarrage Rapide

### Option 1: Docker (Recommandé)

Le moyen le plus rapide de commencer est d'utiliser l'image Docker pré-construite:

```bash
# Télécharger la dernière image
docker pull ghcr.io/soulteary/warden:latest

# Créer un fichier de données
cat > data.json <<EOF
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
EOF

# Exécuter le conteneur
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Astuce**: Pour des exemples complets avec Docker Compose, consultez le [Répertoire d'Exemples](example/README.md).

### Option 2: À partir du Code Source

1. **Cloner et construire**
```bash
git clone <repository-url>
cd warden
go mod download
```

2. **Créer un fichier de données**
Créez un fichier `data.json` (référez-vous à `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

3. **Exécuter le service**
```bash
go run main.go --api-key your-api-key-here
```

## ⚙️ Configuration Essentielle

Warden prend en charge la configuration via les arguments de ligne de commande, les variables d'environnement et les fichiers de configuration. Voici les paramètres les plus essentiels:

| Paramètre | Variable d'Environnement | Description | Requis |
|-----------|-------------------------|-------------|--------|
| Port | `PORT` | Port du serveur HTTP (par défaut: 8081) | Non |
| Clé API | `API_KEY` | Clé d'authentification API (recommandée pour la production) | Recommandé |
| Redis | `REDIS` | Adresse Redis pour la mise en cache et les verrous distribués (ex: `localhost:6379`) | Optionnel |
| Fichier de Données | - | Chemin du fichier de données local (par défaut: `data.json`) | Oui* |
| Configuration Distante | `CONFIG` | URL de l'API distante pour la récupération de données | Optionnel |

\* Requis si aucune API distante n'est utilisée

Pour les options de configuration complètes, consultez la [Documentation de Configuration](docs/enUS/CONFIGURATION.md).

## 📡 Utilisation de l'API

Warden fournit une API RESTful pour interroger les listes d'utilisateurs, la pagination et les vérifications de santé. Le service prend en charge les réponses multi-langues via le paramètre de requête `?lang=xx` ou l'en-tête `Accept-Language`.

**Exemple**:
```bash
# Interroger les utilisateurs
curl -H "X-API-Key: your-key" "http://localhost:8081/"

# Vérification de santé
curl "http://localhost:8081/healthz"
```

Pour la documentation API complète, consultez la [Documentation API](docs/enUS/API.md) ou la [Spécification OpenAPI](openapi.yaml).

## 📊 Performance

Basé sur le test de charge wrk (30s, 16 threads, 100 connexions):
- **Requêtes/seconde**: 5038.81
- **Latence Moyenne**: 21.30ms
- **Latence Maximale**: 226.09ms

## 📚 Documentation

### Documentation Principale

- **[Architecture](docs/enUS/ARCHITECTURE.md)** - Architecture technique et décisions de conception
- **[Référence API](docs/enUS/API.md)** - Documentation complète des points de terminaison API
- **[Configuration](docs/enUS/CONFIGURATION.md)** - Référence et exemples de configuration
- **[Déploiement](docs/enUS/DEPLOYMENT.md)** - Guide de déploiement (Docker, Kubernetes, etc.)

### Ressources Supplémentaires

- **[Guide de Développement](docs/enUS/DEVELOPMENT.md)** - Configuration de l'environnement de développement et guide de contribution
- **[Sécurité](docs/enUS/SECURITY.md)** - Fonctionnalités de sécurité et meilleures pratiques
- **[SDK](docs/enUS/SDK.md)** - Documentation d'utilisation du SDK Go
- **[Exemples](example/README.md)** - Exemples de démarrage rapide (de base et avancés)

## 📄 Licence

Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 🤝 Contribution

Les soumissions d'Issues et de Pull Requests sont les bienvenues! Consultez [CONTRIBUTING.md](docs/enUS/CONTRIBUTING.md) pour les directives.
