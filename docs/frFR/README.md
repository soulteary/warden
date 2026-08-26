# Index de Documentation

Bienvenue dans la documentation du service de données utilisateur Warden AllowList.

## 🌐 Documentation Multilingue

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 Liste des Documents

### Documents Principaux

- **[README.md](../../README.frFR.md)** - Vue d'ensemble du projet et guide de démarrage rapide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architecture technique et décisions de conception

### Documents Détaillés

- **[API.md](API.md)** - Documentation complète des points de terminaison API
  - Points de terminaison de requête de liste d'utilisateurs
  - Fonctionnalité de pagination
  - Points de terminaison de vérification de santé
  - Formats de réponse d'erreur

- **[CONFIGURATION.md](CONFIGURATION.md)** - Référence de configuration
  - Méthodes de configuration
  - Éléments de configuration requis
  - Éléments de configuration optionnels
  - Stratégies de fusion de données
  - Exemples de configuration
  - Meilleures pratiques de configuration

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Guide de déploiement
  - Déploiement Docker (y compris les images GHCR)
  - Déploiement Docker Compose
  - Déploiement local
  - Déploiement en environnement de production
  - Déploiement Kubernetes
  - Optimisation des performances

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Guide de développement
  - Configuration de l'environnement de développement
  - Explication de la structure du code
  - Guide de test
  - Guide de contribution

- **[SDK.md](SDK.md)** - Documentation d'utilisation du SDK
  - Installation et utilisation du SDK Go
  - Description de l'interface API
  - Code d'exemple

- **[SECURITY.md](SECURITY.md)** - Documentation de sécurité
  - Fonctionnalités de sécurité
  - Configuration de sécurité
  - Meilleures pratiques

- **[CODE_STYLE.md](CODE_STYLE.md)** - Guide de style de code
  - Standards de code
  - Conventions de nommage
  - Meilleures pratiques

## 🌐 Support Multilingue

Warden prend en charge une fonctionnalité d'internationalisation (i18N) complète. Toutes les réponses API, les messages d'erreur et les journaux prennent en charge l'internationalisation.

### Langues Prises en Charge

- 🇺🇸 Anglais (en) - Langue par défaut
- 🇨🇳 Chinois (zh)
- 🇫🇷 Français (fr)
- 🇮🇹 Italien (it)
- 🇯🇵 Japonais (ja)
- 🇩🇪 Allemand (de)
- 🇰🇷 Coréen (ko)

### Détection de Langue

Warden prend en charge deux méthodes de détection de langue avec la priorité suivante :

1. **Paramètre de Requête**: Spécifier la langue via le paramètre de requête URL `?lang=fr`
2. **En-tête Accept-Language**: Détection automatique de la préférence de langue du navigateur ou du client
3. **Langue par Défaut**: Anglais si non spécifié

### Exemples d'Utilisation

#### Spécifier la Langue via le Paramètre de Requête

```bash
# Utiliser le français
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"

# Utiliser le japonais
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"

# Utiliser l'allemand
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=de"
```

#### Détection Automatique via l'En-tête Accept-Language

```bash
# Le navigateur envoie automatiquement l'en-tête Accept-Language
curl -H "X-API-Key: your-key" \
     -H "Accept-Language: fr-FR,fr;q=0.9,en;q=0.8" \
     "http://localhost:8081/"
```

### Portée de l'Internationalisation

Le contenu suivant prend en charge plusieurs langues :

- ✅ Messages de réponse d'erreur API
- ✅ Messages d'erreur de code d'état HTTP
- ✅ Messages de journal (basés sur le contexte de la requête)
- ✅ Messages de configuration et d'avertissement

### Implémentation Technique

- Utilise le contexte de requête pour stocker les informations de langue, évite l'état global
- Prend en charge le changement de langue thread-safe
- Retour automatique à l'anglais (si la traduction n'est pas trouvée)
- Toutes les traductions sont intégrées dans le code, aucun fichier externe requis

### Notes de Développement

Pour ajouter de nouvelles traductions ou modifier les traductions existantes, veuillez modifier la map `translations` dans le fichier `internal/i18n/i18n.go`.

## 🚀 Navigation Rapide

### Pour Commencer

1. Lisez [README.frFR.md](../../README.frFR.md) pour comprendre le projet
2. Consultez la section [Démarrage Rapide](../../README.frFR.md#démarrage-rapide)
3. Référez-vous à [Configuration](../../README.frFR.md#configuration) pour configurer le service

### Développeurs

1. Lisez [ARCHITECTURE.md](ARCHITECTURE.md) pour comprendre l'architecture
2. Consultez [API.md](API.md) pour comprendre les interfaces API
3. Référez-vous au [Guide de Développement](../../README.frFR.md#guide-de-développement) pour le développement

### Opérations

1. Lisez [DEPLOYMENT.md](DEPLOYMENT.md) pour comprendre les méthodes de déploiement
2. Consultez [CONFIGURATION.md](CONFIGURATION.md) pour comprendre les options de configuration
3. Référez-vous à [Optimisation des Performances](DEPLOYMENT.md#optimisation-des-performances) pour optimiser le service

## 📖 Structure des Documents

```
warden/
├── README.md              # Document principal du projet (Français)
├── README.frFR.md         # Document principal du projet (Français)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Index de documentation (Anglais)
│   │   ├── ARCHITECTURE.md # Document d'architecture (Anglais)
│   │   ├── API.md          # Document API (Anglais)
│   │   ├── CONFIGURATION.md # Référence de configuration (Anglais)
│   │   ├── DEPLOYMENT.md   # Guide de déploiement (Anglais)
│   │   ├── DEVELOPMENT.md  # Guide de développement (Anglais)
│   │   ├── SDK.md          # Document SDK (Anglais)
│   │   ├── SECURITY.md     # Document de sécurité (Anglais)
│   │   └── CODE_STYLE.md   # Style de code (Anglais)
│   └── frFR/
│       ├── README.md       # Index de documentation (Français, ce fichier)
│       ├── ARCHITECTURE.md # Document d'architecture (Français)
│       ├── API.md          # Document API (Français)
│       ├── CONFIGURATION.md # Référence de configuration (Français)
│       ├── DEPLOYMENT.md   # Guide de déploiement (Français)
│       ├── DEVELOPMENT.md  # Guide de développement (Français)
│       ├── SDK.md          # Document SDK (Français)
│       ├── SECURITY.md     # Document de sécurité (Français)
│       └── CODE_STYLE.md   # Style de code (Français)
└── ...
```

## 🔍 Recherche par Sujet

### Lié à la Configuration

- Configuration des variables d'environnement: [CONFIGURATION.md](CONFIGURATION.md)
- Stratégies de fusion de données: [CONFIGURATION.md](CONFIGURATION.md)
- Exemples de configuration: [CONFIGURATION.md](CONFIGURATION.md)

### Lié à l'API

- Liste des points de terminaison API: [API.md](API.md)
- Gestion des erreurs: [API.md](API.md)
- Fonctionnalité de pagination: [API.md](API.md)

### Lié au Déploiement

- Déploiement Docker: [DEPLOYMENT.md#déploiement-docker](DEPLOYMENT.md#déploiement-docker)
- Images GHCR: [DEPLOYMENT.md#utilisation-dimage-préconstruite-recommandé](DEPLOYMENT.md#utilisation-dimage-préconstruite-recommandé)
- Environnement de production: [DEPLOYMENT.md#déploiement-environnement-de-production-recommandations](DEPLOYMENT.md#déploiement-environnement-de-production-recommandations)
- Kubernetes: [DEPLOYMENT.md#déploiement-kubernetes](DEPLOYMENT.md#déploiement-kubernetes)

### Lié à l'Architecture

- Pile technologique: [ARCHITECTURE.md](ARCHITECTURE.md)
- Structure du projet: [ARCHITECTURE.md](ARCHITECTURE.md)
- Composants principaux: [ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 Recommandations d'Utilisation

1. **Utilisateurs pour la première fois**: Commencez par [README.frFR.md](../../README.frFR.md) et suivez le guide de démarrage rapide
2. **Configurer le service**: Référez-vous à [CONFIGURATION.md](CONFIGURATION.md) pour comprendre toutes les options de configuration
3. **Déployer le service**: Consultez [DEPLOYMENT.md](DEPLOYMENT.md) pour comprendre les méthodes de déploiement
4. **Développer des extensions**: Lisez [ARCHITECTURE.md](ARCHITECTURE.md) pour comprendre la conception de l'architecture
5. **Intégrer le SDK**: Référez-vous à [SDK.md](SDK.md) pour apprendre à utiliser le SDK

## 📝 Mises à Jour des Documents

La documentation est continuellement mise à jour au fur et à mesure de l'évolution du projet. Si vous trouvez des erreurs ou avez besoin d'ajouts, veuillez soumettre un Issue ou une Pull Request.

## 🤝 Contribution

Les améliorations de la documentation sont les bienvenues :

1. Trouver des erreurs ou des domaines à améliorer
2. Soumettre un Issue décrivant le problème
3. Ou soumettre directement une Pull Request
