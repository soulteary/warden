# Documentation de sécurité

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

Ce document présente les fonctionnalités de sécurité de Warden, sa configuration de sécurité et les bonnes pratiques associées.

## Fonctionnalités de sécurité mises en œuvre

1. **Authentification de l'API** : prend en charge l'authentification par clé d'API pour protéger les points de terminaison sensibles
2. **Protection SSRF** : valide strictement les URL de configuration distante afin d'éviter les attaques par falsification de requête côté serveur
3. **Validation des entrées** : valide strictement tous les paramètres d'entrée afin d'éviter les attaques par injection
4. **Limitation de débit** : limitation par adresse IP afin de prévenir les attaques DDoS
5. **Vérification TLS** : les environnements de production imposent la vérification des certificats TLS
6. **Gestion des erreurs** : les environnements de production masquent les informations d'erreur détaillées afin d'éviter toute fuite d'informations
7. **En-têtes de réponse de sécurité** : ajoute automatiquement les en-têtes HTTP liés à la sécurité
8. **Liste d'autorisation IP** : permet de configurer une liste d'autorisation IP pour les points de terminaison de contrôle de santé
9. **Validation du fichier de configuration** : empêche les attaques par traversée de chemin
10. **Limites de taille JSON** : limite la taille du corps des réponses JSON afin de prévenir les attaques par épuisement mémoire
11. **Limite de longueur des paramètres de requête utilisateur** : un paramètre isolé (`phone`/`mail`/`user_id`) ne doit pas dépasser 512 octets, afin d'éviter un déni de service et l'engorgement des journaux et du cache
12. **Assainissement des données personnelles dans les journaux d'audit** : l'identifiant écrit dans l'audit est masqué pour phone/mail, afin d'éviter toute exposition de données personnelles si le stockage d'audit est compromis

## Bonnes pratiques de sécurité

### 1. Configuration de l'environnement de production

**Configuration obligatoire** :
- `ENVIRONMENT=production` **doit** être défini pour activer le durcissement de production.
- Au moins un mécanisme d'authentification de service **doit** être configuré : `API_KEY`, HMAC v2 ou mTLS.
- `TRUSTED_PROXY_IPS` **doit** être configuré pour obtenir correctement l'IP du client
- `HEALTH_CHECK_IP_WHITELIST` **doit** restreindre l'accès au contrôle de santé (ou `/health` et `/healthcheck` doivent être restreints au niveau du réseau ou du reverse proxy)
- `/metrics` **doit** être restreint. Avec `ENVIRONMENT=production`, l'authentification est requise par défaut ; conservez ce réglage (ou restreignez le chemin au niveau du reverse proxy ou du réseau) et ne définissez **pas** `WARDEN_METRICS_REQUIRE_AUTH=false` en production.

**Exemple de configuration** :
```bash
export API_KEY="your-strong-api-key-here"
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. Gestion des informations sensibles

**Pratiques recommandées** :
- ✅ Stocker les mots de passe et les clés dans des variables d'environnement
- ✅ Utiliser des fichiers de mot de passe (`REDIS_PASSWORD_FILE`) pour les mots de passe Redis
- ✅ Utiliser des espaces réservés ou des commentaires dans les fichiers de configuration
- ✅ Vérifier que les permissions des fichiers de configuration sont correctes (par ex. `chmod 600`)

**Déconseillé** :
- ❌ Coder en dur les mots de passe dans les fichiers de configuration
- ❌ Transmettre les mots de passe via les arguments de ligne de commande (ils apparaissent dans la liste des processus)
- ❌ Committer dans le système de gestion de versions des fichiers de configuration contenant des informations sensibles

**Exemple** :
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # Utiliser la variable d'environnement REDIS_PASSWORD ou REDIS_PASSWORD_FILE

app:
  # api_key: ""  # Utiliser la variable d'environnement API_KEY
```

### 3. Sécurité réseau

**Configuration obligatoire** :
- Les environnements de production doivent utiliser HTTPS
- Configurer des règles de pare-feu pour restreindre l'accès
- Mettre régulièrement à jour les dépendances pour corriger les vulnérabilités connues

**Configuration recommandée** :
- Utiliser un reverse proxy (comme Nginx) pour gérer SSL/TLS
- Configurer `TRUSTED_PROXY_IPS` pour obtenir correctement l'IP réelle du client
- Utiliser des mots de passe et des clés d'API robustes
- Désactiver `HTTP_INSECURE_TLS` (doit valoir `false` en production)

### 4. Supervision et audit

**Pratiques recommandées** :
- Superviser les journaux d'événements de sécurité
- Examiner régulièrement les journaux d'accès
- Utiliser des outils d'analyse de sécurité dans la CI/CD
- Mettre en place des mécanismes d'alerte

**Gestion du niveau de journalisation** :
- En production, le niveau `info` ou `warn` est recommandé
- Toutes les opérations de modification du niveau de journalisation sont enregistrées dans les journaux d'audit de sécurité
- Le niveau de journalisation peut être ajusté dynamiquement via l'API `/log/level` (authentification par clé d'API requise)

## Sécurité de l'API

### Authentification par clé d'API

Certains points de terminaison exigent une authentification par clé d'API :

**Points de terminaison exigeant une authentification** :
- `GET /` – Obtenir la liste des utilisateurs
- `GET /user` – Interroger un utilisateur unique
- `GET /log/level` – Obtenir le niveau de journalisation
- `POST /log/level` – Définir le niveau de journalisation

**Points de terminaison sans authentification** (doivent être protégés autrement en production) :
- `GET /health` – Contrôle de santé (`HEALTH_CHECK_IP_WHITELIST` ou une isolation réseau **doit** être configurée)
- `GET /healthcheck` – Contrôle de santé (idem)
- `GET /metrics` – Métriques Prometheus (une clé d'API **doit** être définie pour la collecte, ou l'accès restreint au niveau du reverse proxy ou du réseau ; ne pas exposer publiquement)

**Méthodes d'authentification** :
1. **En-tête X-API-Key** :
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **En-tête Authorization Bearer** :
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Limitation de débit

Par défaut, les requêtes d'API sont protégées par une limitation de débit :

- **Limite** : 60 requêtes par minute
- **Fenêtre** : 1 minute
- **En cas de dépassement** : renvoie `429 Too Many Requests`

Peut être ajustée via le fichier de configuration :

```yaml
rate_limit:
  rate: 60  # Requêtes par minute
  window: 1m
```

### Liste d'autorisation IP

Deux types de listes d'autorisation IP sont pris en charge :

1. **Liste d'autorisation IP globale** (`IP_WHITELIST`) :
   - Restreint l'accès à tous les points de terminaison
   - Prend en charge le format de plage CIDR

2. **Liste d'autorisation IP du contrôle de santé** (`HEALTH_CHECK_IP_WHITELIST`) :
   - Restreint uniquement les points de terminaison `/health` et `/healthcheck`
   - Prend en charge le format de plage CIDR

**Exemple de configuration** :
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Sécurité des données

### Sécurité de l'API de configuration distante

- Les API de configuration distante doivent utiliser des mécanismes d'authentification (en-tête Authorization)
- Le protocole HTTPS est recommandé
- Vérifier les certificats TLS de l'API distante (obligatoire en production)

### Sécurité de Redis

- Redis doit être configuré avec une protection par mot de passe
- Utiliser les variables d'environnement `REDIS_PASSWORD` ou `REDIS_PASSWORD_FILE`
- Restreindre l'accès réseau à Redis (autoriser uniquement le serveur applicatif)
- Mettre régulièrement à jour Redis pour corriger les vulnérabilités connues

### Sécurité du fichier de données

- Vérifier que les permissions du fichier `data.json` sont correctes
- Ne pas committer de données sensibles dans le système de gestion de versions
- Sauvegarder régulièrement les fichiers de données

## En-têtes de réponse de sécurité

Warden ajoute automatiquement les en-têtes HTTP de sécurité suivants :

- `X-Content-Type-Options: nosniff` – Empêche le reniflage de type MIME
- `X-Frame-Options: DENY` – Empêche le détournement de clic
- `X-XSS-Protection: 1; mode=block` – Protection XSS

## Gestion des erreurs

### Mode production

En mode production (`ENVIRONMENT=production`) :

- Les informations d'erreur détaillées sont masquées afin d'éviter toute fuite d'informations
- Des messages d'erreur génériques sont renvoyés
- Les informations d'erreur détaillées ne sont consignées que dans les journaux

### Mode développement

En mode développement :

- Les informations d'erreur détaillées sont affichées pour faciliter le débogage
- Les informations de pile d'appels sont incluses

## Audit de sécurité

Pour les consignes de durcissement et de vérification des versions publiées, voir [Release Security](../RELEASE_SECURITY.md).

## Signalement de vulnérabilités

Si vous découvrez une vulnérabilité de sécurité, veuillez la signaler ainsi :

1. Créez une Issue de sécurité privée (si cette option est prise en charge)
2. Envoyez un e-mail aux mainteneurs du projet
3. Ne divulguez pas publiquement la vulnérabilité avant qu'elle ne soit corrigée

## Authentification entre services (facultatif)

Si vous choisissez de vous intégrer à d'autres services (comme Stargate), l'authentification entre services permet d'assurer la sécurité. **mTLS et HMAC sont implémentés** ; l'ordre de priorité est **mTLS > HMAC > clé d'API**. Warden prend en charge les méthodes suivantes :

**Remarque** : si Warden est utilisé de manière autonome, l'authentification entre services est facultative.

### mTLS (recommandé)

Utiliser des certificats TLS mutuels pour l'authentification, ce qui offre un niveau de sécurité supérieur.

**Configuration** :

1. **Générer les certificats** :
   ```bash
   # Générer le certificat de l'autorité de certification
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # Générer le certificat serveur de Warden
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # Générer le certificat client de Stargate
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Configuration de Warden** (variables d'environnement) :
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Configuration de Stargate** :
   - Configurer le chemin du certificat client
   - Configurer le chemin du certificat de l'autorité de certification pour vérifier le certificat serveur de Warden

### Signature HMAC

Utiliser une signature HMAC-SHA256 pour vérifier les requêtes ; ce mécanisme est plus simple à déployer.

**Algorithme de signature** :
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**En-têtes de requête** :
- `X-Signature` : valeur de la signature HMAC
- `X-Timestamp` : horodatage Unix (secondes)
- `X-Key-Id` : identifiant de clé (inclus dans la signature, pour une rotation de clés sûre)
- `X-Nonce` : nonce unique de 128 bits en hexadécimal
- `X-Signature-Version`: `v2`

**Configuration de Warden** (variables d'environnement) :
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # Tolérance d'horodatage (secondes), 60 par défaut
export WARDEN_HMAC_ALLOW_V1=false          # Valeur par défaut ; à passer à true uniquement pendant une migration héritée limitée dans le temps
```

La configuration de production exige que chaque secret HMAC comporte au moins 32 octets bruts.
Générez les secrets avec une source aléatoire cryptographiquement sûre et stockez-les dans un
gestionnaire de secrets ; ne réutilisez pas la valeur d'illustration ci-dessus.

**Exemple avec le SDK Go** :
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**Règles de vérification** :
- Warden vérifie que l'horodatage se situe dans la plage de tolérance (±60 secondes par défaut)
- Warden vérifie tous les champs canoniques, y compris l'identifiant de clé, et rejette les nonces réutilisés
- Si la vérification de la signature échoue, renvoie `401 Unauthorized`

### Priorité de configuration

1. **mTLS** : si des certificats TLS sont configurés, mTLS est utilisé en premier
2. **HMAC** : si mTLS n'est pas configuré, la signature HMAC est utilisée
3. **Clé d'API** : si aucun des deux n'est configuré, repli sur l'authentification par clé d'API (déconseillé pour les appels entre services)

### Recommandations de sécurité

1. **Environnement de production** : l'utilisation de mTLS pour l'authentification entre services est fortement recommandée
2. **Gestion des clés** : utilisez un service de gestion de clés (comme HashiCorp Vault) pour stocker clés et certificats
3. **Rotation des clés** : renouvelez régulièrement les clés HMAC et les certificats TLS
4. **Isolation réseau** : dans la mesure du possible, utilisez des politiques réseau pour n'autoriser l'accès à Warden que depuis Stargate

## Documentation associée

- [Documentation de configuration](CONFIGURATION.md) – Découvrez les options de configuration liées à la sécurité
- [Documentation de déploiement](DEPLOYMENT.md) – Découvrez les recommandations de déploiement en production
- [Documentation de l'API](API.md) – Découvrez les fonctionnalités de sécurité de l'API
- [Documentation d'architecture](ARCHITECTURE.md) – Découvrez l'architecture d'intégration des services
