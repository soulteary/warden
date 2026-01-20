# Warden

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

Un service de données utilisateur de liste d'autorisation (AllowList) haute performance qui prend en charge la synchronisation et la fusion de données à partir de sources de configuration locales et distantes.

![Warden](.github/assets/banner.jpg)

> **Warden** (Le Gardien) — Le gardien de la Porte des Étoiles qui décide qui peut passer et qui sera refusé. Tout comme le Gardien de Stargate garde la Porte des Étoiles, Warden garde votre liste d'autorisation, garantissant que seuls les utilisateurs autorisés peuvent passer.

## 📋 Aperçu du Projet

Warden est un service API HTTP léger développé en Go, principalement utilisé pour fournir et gérer les données utilisateur de liste d'autorisation (numéros de téléphone et adresses e-mail). Le service prend en charge la récupération de données à partir de fichiers de configuration locaux et d'API distantes, et fournit plusieurs stratégies de fusion de données pour assurer la performance et la fiabilité des données en temps réel.

## ✨ Fonctionnalités Principales

- 🚀 **Haute Performance**: Prend en charge plus de 5000 requêtes par seconde avec une latence moyenne de 21ms
- 🔄 **Sources de Données Multiples**: Prend en charge les fichiers de configuration locaux et les API distantes
- 🎯 **Stratégies Flexibles**: Fournit 6 modes de fusion de données (priorité distante, priorité locale, distant uniquement, local uniquement, etc.)
- ⏰ **Mises à Jour Planifiées**: Tâches planifiées basées sur des verrous distribués Redis pour la synchronisation automatique des données
- 📦 **Déploiement Conteneurisé**: Support Docker complet, prêt à l'emploi
- 📊 **Journalisation Structurée**: Utilise zerolog pour fournir des journaux d'accès et d'erreur détaillés
- 🔒 **Verrous Distribués**: Utilise Redis pour s'assurer que les tâches planifiées ne s'exécutent pas de manière répétée dans les environnements distribués
- 🌐 **Support Multi-langues**: Prend en charge 7 langues (Anglais, Chinois, Français, Italien, Japonais, Allemand, Coréen) avec détection automatique de la langue préférée

## 🏗️ Conception de l'Architecture

Warden utilise une conception d'architecture en couches, comprenant la couche HTTP, la couche métier et la couche d'infrastructure. Le système prend en charge plusieurs sources de données, la mise en cache multi-niveaux et les mécanismes de verrouillage distribués.

Pour la documentation détaillée de l'architecture, veuillez vous référer à: [Documentation de Conception de l'Architecture](docs/enUS/ARCHITECTURE.md)

## 📦 Installation et Exécution

> 💡 **Démarrage Rapide**: Vous voulez découvrir rapidement Warden ? Consultez nos [Exemples de Démarrage Rapide](example/README.en.md):
> - [Exemple Simple](example/basic/README.en.md) - Utilisation de base, fichier de données local uniquement
> - [Exemple Avancé](example/advanced/README.en.md) - Fonctionnalités complètes, incluant l'API distante et le service Mock

### Prérequis

- Go 1.25+ (référez-vous à [go.mod](go.mod))
- Redis (pour les verrous distribués et la mise en cache)
- Docker (optionnel, pour le déploiement conteneurisé)

### Démarrage Rapide

1. **Cloner le projet**
```bash
git clone <repository-url>
cd warden
```

2. **Installer les dépendances**
```bash
go mod download
```

3. **Configurer le fichier de données local**
Créez un fichier `data.json` (référez-vous à `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

4. **Exécuter le service**
```bash
go run main.go
```

Pour les instructions détaillées de configuration et de déploiement, veuillez vous référer à:
- [Documentation de Configuration](docs/enUS/CONFIGURATION.md) - Découvrir toutes les options de configuration
- [Documentation de Déploiement](docs/enUS/DEPLOYMENT.md) - Découvrir les méthodes de déploiement

## ⚙️ Configuration

Warden prend en charge plusieurs méthodes de configuration: arguments de ligne de commande, variables d'environnement et fichiers de configuration. Le système fournit 6 modes de fusion de données avec des stratégies de configuration flexibles.

Pour la documentation détaillée de configuration, veuillez vous référer à: [Documentation de Configuration](docs/enUS/CONFIGURATION.md)

## 📡 Documentation API

Warden fournit une API RESTful complète avec support pour les requêtes de liste d'utilisateurs, la pagination, les vérifications de santé, etc. Le projet fournit également une documentation de spécification OpenAPI 3.0.

Pour la documentation API détaillée, veuillez vous référer à: [Documentation API](docs/enUS/API.md)

Fichier de spécification OpenAPI: [openapi.yaml](openapi.yaml)

## 🌐 Support Multi-langues

Warden prend en charge une fonctionnalité complète d'internationalisation (i18N). Toutes les réponses API, messages d'erreur et journaux prennent en charge l'internationalisation.

### Langues Supportées

- 🇺🇸 Anglais (en) - Par défaut
- 🇨🇳 Chinois (zh)
- 🇫🇷 Français (fr)
- 🇮🇹 Italien (it)
- 🇯🇵 Japonais (ja)
- 🇩🇪 Allemand (de)
- 🇰🇷 Coréen (ko)

### Détection de la Langue

Warden prend en charge deux méthodes de détection de langue avec la priorité suivante :

1. **Paramètre de requête**: Spécifier la langue via `?lang=fr`
2. **En-tête Accept-Language**: Détection automatique de la préférence linguistique du navigateur
3. **Langue par défaut**: Anglais si non spécifié

### Exemples d'Utilisation

```bash
# Spécifier le français via le paramètre de requête
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"

# Détection automatique via l'en-tête Accept-Language
curl -H "X-API-Key: your-key" -H "Accept-Language: fr-FR,fr;q=0.9" "http://localhost:8081/"

# Utiliser le chinois
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=zh"
```

## 🔌 Utilisation du SDK

Warden fournit un SDK Go pour faciliter l'intégration dans d'autres projets. Le SDK fournit des interfaces API simples avec support pour la mise en cache, l'authentification, etc.

Pour la documentation SDK détaillée, veuillez vous référer à: [Documentation SDK](docs/enUS/SDK.md)

## 🐳 Déploiement Docker

Warden prend en charge le déploiement Docker et Docker Compose complet, prêt à l'emploi.

### Démarrage Rapide avec Image Pré-construite (Recommandé)

Utilisez l'image pré-construite fournie par GitHub Container Registry (GHCR) pour démarrer rapidement sans construction locale:

```bash
# Télécharger l'image de la dernière version
docker pull ghcr.io/soulteary/warden:latest

# Exécuter le conteneur (exemple de base)
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Astuce**: L'utilisation d'images pré-construites vous permet de démarrer rapidement sans environnement de construction local. Les images sont automatiquement mises à jour pour garantir que vous utilisez la dernière version.

### Utilisation de Docker Compose

> 🚀 **Déploiement Rapide**: Consultez le [Répertoire d'Exemples](example/README.en.md) pour des exemples de configuration Docker Compose complets

Pour la documentation de déploiement détaillée, veuillez vous référer à: [Documentation de Déploiement](docs/enUS/DEPLOYMENT.md)

## 📊 Métriques de Performance

Basé sur les résultats des tests de charge wrk (test de 30 secondes, 16 threads, 100 connexions):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Latence Moyenne: 21.30ms
Latence Maximale: 226.09ms
```

## 📁 Structure du Projet

```
warden/
├── main.go                 # Point d'entrée du programme
├── data.example.json      # Exemple de fichier de données local
├── config.example.yaml    # Exemple de fichier de configuration
├── openapi.yaml           # Fichier de spécification OpenAPI
├── go.mod                 # Définition du module Go
├── docker-compose.yml     # Configuration Docker Compose
├── LICENSE                # Fichier de licence
├── README.*.md            # Documents du projet multilingues (Chinois/Anglais/Français/Italien/Japonais/Allemand/Coréen)
├── CONTRIBUTING.*.md      # Guides de contribution multilingues
├── docker/
│   └── Dockerfile         # Fichier de construction d'image Docker
├── docs/                  # Répertoire de documentation (multilingue)
│   ├── enUS/              # Documentation anglaise
│   └── zhCN/              # Documentation chinoise
├── example/               # Exemples de démarrage rapide
│   ├── basic/             # Exemple simple (fichier local uniquement)
│   └── advanced/          # Exemple avancé (fonctionnalités complètes, inclut Mock API)
├── internal/
│   ├── cache/             # Implémentation du cache et des verrous Redis
│   ├── cmd/               # Analyse des arguments de ligne de commande
│   ├── config/            # Gestion de la configuration
│   ├── define/            # Définitions de constantes et structures de données
│   ├── di/                # Injection de dépendances
│   ├── errors/            # Gestion des erreurs
│   ├── i18n/              # Support d'internationalisation
│   ├── logger/            # Initialisation de la journalisation
│   ├── metrics/           # Collecte de métriques
│   ├── middleware/        # Middleware HTTP
│   ├── parser/            # Analyseur de données (local/distant)
│   ├── router/            # Gestion des routes HTTP
│   ├── validator/         # Validateur
│   └── version/           # Informations de version
├── pkg/
│   ├── gocron/            # Planificateur de tâches planifiées
│   └── warden/            # SDK Warden
├── scripts/               # Répertoire de scripts
└── .github/               # Configuration GitHub (CI/CD, modèles Issue/PR, etc.)
```

## 🔒 Fonctionnalités de Sécurité

Warden implémente plusieurs fonctionnalités de sécurité, notamment l'authentification API, la protection SSRF, la limitation du débit, la vérification TLS, etc.

Pour la documentation de sécurité détaillée, veuillez vous référer à: [Documentation de Sécurité](docs/enUS/SECURITY.md)

## 🔧 Guide de Développement

> 📚 **Exemples de Référence**: Consultez le [Répertoire d'Exemples](example/README.en.md) pour des exemples de code et de configurations complets pour différents scénarios d'utilisation.

Pour la documentation de développement détaillée, veuillez vous référer à: [Documentation de Développement](docs/enUS/DEVELOPMENT.md)

### Standards de Code

Le projet suit les standards de code officiels de Go et les meilleures pratiques. Pour les standards détaillés, veuillez vous référer à:

- [CODE_STYLE.md](docs/enUS/CODE_STYLE.md) - Guide de style de code
- [CONTRIBUTING.en.md](CONTRIBUTING.en.md) - Guide de contribution

## 📄 Licence

Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 🤝 Contribution

Les soumissions d'Issues et de Pull Requests sont les bienvenues !

## 📞 Contact

Pour les questions ou suggestions, veuillez contacter via Issues.

---

**Version**: Le programme affiche la version, l'heure de construction et la version du code au démarrage (via `warden --version` ou les journaux de démarrage)
