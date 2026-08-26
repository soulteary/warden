# Documentation de Sécurité

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Ce document explique les fonctionnalités de sécurité de Warden, la configuration de sécurité et les meilleures pratiques.


## Fonctionnalités de Sécurité Implémentées

1. **Authentification API**: Prend en charge l'authentification par clé API pour protéger les points de terminaison sensibles
2. **Protection SSRF**: Valide strictement les URL de configuration distantes pour prévenir les attaques de falsification de requête côté serveur
3. **Validation des Entrées**: Valide strictement tous les paramètres d'entrée pour prévenir les attaques par injection
4. **Limitation du Débit**: Limitation du débit basée sur l'IP pour prévenir les attaques DDoS
5. **Vérification TLS**: Les environnements de production appliquent la vérification des certificats TLS
6. **Gestion des Erreurs**: Les environnements de production masquent les informations d'erreur détaillées pour prévenir les fuites d'informations
7. **En-têtes de Réponse de Sécurité**: Ajoute automatiquement les en-têtes de réponse HTTP liés à la sécurité
8. **Liste Blanche IP**: Prend en charge la configuration de la liste blanche IP pour les points de terminaison de vérification de santé
9. **Validation des Fichiers de Configuration**: Empêche les attaques de traversée de chemin
10. **Limites de Taille JSON**: Limite la taille du corps de réponse JSON pour prévenir les attaques d'épuisement de la mémoire

## Meilleures Pratiques de Sécurité

### 1. Configuration de l'Environnement de Production

**Configuration Requise**:
- Doit définir la variable d'environnement `API_KEY`
- Définir `MODE=production` pour activer le mode production
- Configurer `TRUSTED_PROXY_IPS` pour obtenir correctement l'IP du client
- Utiliser `HEALTH_CHECK_IP_WHITELIST` pour restreindre l'accès à la vérification de santé

### 2. Gestion des Informations Sensibles

**Pratiques Recommandées**:
- ✅ Utiliser des variables d'environnement pour stocker les mots de passe et les clés
- ✅ Utiliser des fichiers de mot de passe (`REDIS_PASSWORD_FILE`) pour stocker les mots de passe Redis
- ✅ Utiliser des espaces réservés ou des commentaires dans les fichiers de configuration
- ✅ S'assurer que les permissions des fichiers de configuration sont définies correctement (par exemple, `chmod 600`)

### 3. Sécurité Réseau

**Configuration Requise**:
- Les environnements de production doivent utiliser HTTPS
- Configurer les règles de pare-feu pour restreindre l'accès
- Mettre à jour régulièrement les dépendances pour corriger les vulnérabilités connues

## Sécurité API

### Authentification par Clé API

Certains points de terminaison API nécessitent une authentification par clé API.

**Méthodes d'Authentification**:
1. **En-tête X-API-Key**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **En-tête Authorization Bearer**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Limitation du Débit

Par défaut, les requêtes API sont protégées par une limitation du débit :
- **Limite**: 60 requêtes par minute
- **Fenêtre**: 1 minute
- **Dépassement**: Retourne `429 Too Many Requests`

## Signalement de Vulnérabilité

Si vous découvrez une vulnérabilité de sécurité, veuillez la signaler via :

1. **GitHub Security Advisory** (Préféré)
   - Allez dans l'onglet [Security](https://github.com/soulteary/warden/security) du dépôt
   - Cliquez sur "Report a vulnerability"
   - Remplissez le formulaire de conseil de sécurité

2. **Email** (Si GitHub Security Advisory n'est pas disponible)
   - Envoyez un email aux mainteneurs du projet
   - Incluez une description détaillée de la vulnérabilité

## Documentation Associée

- [Documentation de Configuration](CONFIGURATION.md) - En savoir plus sur les options de configuration liées à la sécurité
- [Documentation de Déploiement](DEPLOYMENT.md) - En savoir plus sur les recommandations de déploiement en environnement de production
- [Documentation API](API.md) - En savoir plus sur les fonctionnalités de sécurité de l'API
