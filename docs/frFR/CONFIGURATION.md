# Configuration

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

Ce document présente en détail les options de configuration de Warden : modes d'exécution, formats des fichiers de configuration, variables d'environnement, etc.

**Priorité de configuration** : arguments de ligne de commande > variables d'environnement > fichier de configuration (YAML) > valeurs par défaut.

Pour un **tableau complet des options** (chemins YAML, variables d'environnement, valeurs par défaut, règles de validation), consultez la [CONFIGURATION zhCN](../zhCN/CONFIGURATION.md). Résumé :

| Catégorie | YAML / Env | Remarques |
|----------|------------|--------|
| Serveur | `server.*` / `PORT` | port, read_timeout, write_timeout, shutdown_timeout, idle_timeout, max_header_bytes |
| Redis | `redis.*` / `REDIS`, `REDIS_PASSWORD`, `REDIS_PASSWORD_FILE`, `REDIS_ENABLED` | addr, password, password_file, db ; Redis activé par défaut (`true`), sauf en ONLY_LOCAL sans REDIS |
| Cache | `cache.ttl`, `cache.update_interval` | aucun remplacement par variable d'environnement ; update_interval par défaut 5s |
| Limitation de débit | `rate_limit.rate`, `rate_limit.window` | par défaut 60/min, fenêtre de 1m |
| Client HTTP | `http.*` / `HTTP_TIMEOUT`, `HTTP_MAX_IDLE_CONNS`, `HTTP_INSECURE_TLS` | timeout, max_idle_conns, insecure_tls, max_retries, retry_delay |
| Remote | `remote.*` / `CONFIG`, `KEY`, `MERGE_MODE`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_RSA_PRIVATE_KEY_FILE`, `REMOTE_RSA_PRIVATE_KEY` | url, key, mode, decrypt_enabled, rsa_private_key_file |
| Tâche | `task.interval` | aucun remplacement par variable d'environnement avec un fichier de configuration ; utilisez `INTERVAL` uniquement sans fichier de configuration |
| Application | `app.*` / `API_KEY`, `DATA_FILE`, `DATA_DIR`, `RESPONSE_FIELDS` | mode, api_key, data_file, data_dir, response_fields |
| Traçage | `tracing.enabled`, `tracing.endpoint` / `OTLP_ENABLED`, `OTLP_ENDPOINT` | Avec `--config-file`, `tracing` n'est pas lu depuis ce fichier sauf si `CONFIG_FILE` pointe vers le même chemin |
| Authentification de service | — / `WARDEN_HMAC_KEYS`, `WARDEN_HMAC_TIMESTAMP_TOLERANCE`, `WARDEN_TLS_*` | **Variables d'environnement uniquement** (aucune clé YAML) |
| Santé | — / `SNAPSHOT_MAX_AGE` | Âge maximal accepté pour l'instantané ; durée Go, par défaut `max(30s, 3 × intervalle de tâche)` |

## Mode d'exécution (MERGE_MODE)

Le système prend en charge 7 modes de fusion de données, sélectionnés via `MERGE_MODE` (`MODE` est déprécié) :

| Mode | Description | Cas d'usage |
|------|-------------|----------|
| `DEFAULT` | Comportement historique tolérant, priorité au distant | Compatibilité ascendante pour les déploiements n'ayant jamais choisi de mode |
| `REMOTE_FIRST` | Le distant l'emporte quand le chargement réussit ; une erreur distante est fatale et le dernier instantané valide connu est conservé | Déploiements stricts où le distant fait autorité |
| `ONLY_REMOTE` | Utiliser uniquement la source de données distante | Dépendance totale à la configuration distante |
| `ONLY_LOCAL` | Utiliser uniquement le fichier de configuration local, **Redis désactivé par défaut** (activé si l'adresse `REDIS` est explicitement définie ou si `REDIS_ENABLED=true`) | Environnement hors ligne ou de test |
| `LOCAL_FIRST` | Priorité au local ; les données distantes complètent lorsque les données locales sont absentes | Configuration locale principale, distant en complément |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Priorité au distant, repli sur le local autorisé en cas d'échec distant | Scénarios de haute disponibilité |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Priorité au local, repli sur le distant autorisé en cas d'échec local | Mode hybride |

### Fraîcheur des instantanés et échecs distants

`REMOTE_FIRST` et `ONLY_REMOTE` sont des modes stricts. Si un rafraîchissement distant
planifié échoue, Warden continue de servir le dernier instantané valide connu en mémoire,
enregistre l'échec du rafraîchissement et ne fait pas avancer le `loaded_at` de
l'instantané. Le point de terminaison de santé renvoie HTTP 503 dès que l'instantané
dépasse `SNAPSHOT_MAX_AGE` (par défaut `max(30s, 3 × intervalle de tâche)`). Définissez
la valeur avec une durée Go telle que `2m`.

`REMOTE_FIRST_ALLOW_REMOTE_FAILED` se replie explicitement sur des données locales
validées, marque l'instantané comme `degraded` et reste utilisable avec un HTTP 200.
`LOCAL_FIRST` et `LOCAL_FIRST_ALLOW_REMOTE_FAILED` peuvent rester sains lorsque leur
source locale principale réussit, même si le complément distant est indisponible.
`DEFAULT` conserve le comportement tolérant historique pour des raisons de compatibilité
(y compris cette sémantique de succès local en clair) ; choisissez un mode explicite pour
les nouveaux déploiements de production. Les échecs distants chiffrés qui se replient sur
des données locales sont signalés comme `degraded` dans tous les modes tolérants.

Dans les déploiements multi-réplicas, chaque réplica rafraîchit son propre cache et son
propre instantané locaux au processus. Un verrou Redis distribué n'élit que l'écrivain du
cache Redis partagé ; les non-écrivains continuent donc de faire progresser la fraîcheur
de leur propre instantané.

### Méthodes de configuration

Vous pouvez définir le mode d'exécution des manières suivantes :

**Arguments de ligne de commande** :
```bash
go run . --mode DEFAULT
```

**Variables d'environnement** :
```bash
export MERGE_MODE=DEFAULT
# MODE reste un alias de compatibilité déprécié.
```

**Fichier de configuration** :
```yaml
remote:
  mode: "DEFAULT"
# ou
app:
  mode: "DEFAULT"
```

## Format des fichiers de configuration

### Fichier local de données utilisateur (`data.json`)

Format du fichier local de données utilisateur `data.json` (voir `data.example.json`) :

**Format minimal** (champs obligatoires uniquement) :
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Format complet** (avec tous les champs facultatifs) :
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    }
]
```

**Description des champs** :
- `phone` (obligatoire) : numéro de téléphone de l'utilisateur
- `mail` (obligatoire) : adresse e-mail de l'utilisateur
- `user_id` (facultatif) : identifiant unique de l'utilisateur, généré automatiquement à partir de `phone` ou `mail` s'il n'est pas fourni
- `status` (facultatif) : statut de l'utilisateur ; en l'absence de valeur, le repli sécurisé est `"inactive"`. Définissez explicitement `"active"` pour autoriser l'accès.
- `scope` (facultatif) : tableau des portées de permission de l'utilisateur, tableau vide par défaut
- `role` (facultatif) : rôle de l'utilisateur, chaîne vide par défaut

### Fichier de configuration de l'application (`config.yaml`)

Les fichiers de configuration au format YAML sont pris en charge et se spécifient via le paramètre `--config-file` :

```yaml
server:
  port: "8081"
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 5s
  max_header_bytes: 1048576  # 1 Mo
  idle_timeout: 120s

redis:
  addr: "localhost:6379"
  password: ""  # Préférez la variable d'environnement REDIS_PASSWORD ou REDIS_PASSWORD_FILE
  password_file: ""  # Chemin du fichier de mot de passe (priorité supérieure à password)
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60  # Requêtes par minute
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # Développement uniquement
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # Déchiffrer la réponse distante par RSA (à utiliser avec rsa_private_key_file ou REMOTE_RSA_PRIVATE_KEY)
  rsa_private_key_file: ""    # Chemin du fichier PEM (ou variable REMOTE_RSA_PRIVATE_KEY pour un PEM en ligne)

task:
  interval: 5s

app:
  mode: "DEFAULT"  # Mode de fusion des données ; la politique de production est choisie par ENVIRONMENT
  api_key: ""      # Préférez la variable d'environnement API_KEY
  data_file: "./data.json"
  data_dir: ""     # Facultatif : fusionner tous les *.json d'un répertoire (utilisable avec data_file)
  response_fields: []  # Facultatif : liste blanche des champs de réponse de l'API ; vide = tous les champs

tracing:
  enabled: false
  endpoint: ""     # par ex. "http://localhost:4318"
```

**Priorité de configuration** : arguments de ligne de commande > variables d'environnement > fichier de configuration > valeurs par défaut.

**Remarque sur le traçage** : avec `--config-file`, le programme principal ne lit pas la section `tracing` de ce fichier, sauf si la variable d'environnement `CONFIG_FILE` pointe vers le même chemin, ou si vous utilisez `OTLP_ENABLED` + `OTLP_ENDPOINT`.

Voir le fichier d'exemple : [config.example.yaml](../../config.example.yaml).

## Arguments de ligne de commande

```bash
go run . \
  --port 8081 \                    # Port du service web (par défaut : 8081)
  --redis localhost:6379 \         # Adresse Redis (par défaut : localhost:6379)
  --redis-password "password" \    # Mot de passe Redis (facultatif, variables d'environnement recommandées)
  --redis-enabled=true \           # Activer/désactiver Redis (par défaut : true)
  --config http://example.com/api \ # URL de la configuration distante
  --key "Bearer token" \           # En-tête d'authentification de la configuration distante
  --interval 5 \                   # Intervalle de la tâche planifiée (secondes, par défaut : 5)
  --mode DEFAULT \                 # Mode d'exécution (voir la description ci-dessus)
  --http-timeout 5 \               # Délai d'expiration des requêtes HTTP (secondes, par défaut : 5)
  --http-max-idle-conns 100 \     # Nombre maximal de connexions HTTP inactives (par défaut : 100)
  --http-insecure-tls \           # Ignorer la vérification des certificats TLS (développement uniquement)
  --api-key "your-secret-api-key" \ # Clé d'API pour l'authentification (facultatif, variables d'environnement recommandées)
  --config-file config.yaml        # Chemin du fichier de configuration (format YAML pris en charge)
```

**Remarques** :
- Prise en charge des fichiers de configuration : le paramètre `--config-file` permet de spécifier un fichier de configuration au format YAML
- Sécurité du mot de passe Redis : préférez les variables d'environnement `REDIS_PASSWORD` ou `REDIS_PASSWORD_FILE` aux arguments de ligne de commande
- Vérification des certificats TLS : `--http-insecure-tls` est réservé aux environnements de développement et ne doit pas être utilisé en production

## Variables d'environnement

La configuration par variables d'environnement est prise en charge, avec une priorité inférieure aux arguments de ligne de commande. Pour le tableau complet des options (y compris les règles de validation), consultez la [CONFIGURATION zhCN](../zhCN/CONFIGURATION.md).

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Mot de passe Redis (facultatif)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Chemin du fichier de mot de passe Redis (facultatif ; priorité : REDIS_PASSWORD > REDIS_PASSWORD_FILE > configuration)
export REDIS_ENABLED=true               # Activer/désactiver Redis (facultatif, par défaut : true, accepte true/false/1/0)
                                        # Remarque : en mode ONLY_LOCAL, la valeur par défaut est false
                                        #       Mais si l'adresse REDIS est explicitement définie, Redis est activé automatiquement
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MERGE_MODE=DEFAULT
export DATA_FILE=./data.json          # Chemin du fichier local de données utilisateur
export DATA_DIR=                      # Facultatif : répertoire dont tous les *.json sont fusionnés (utilisable avec DATA_FILE)
export RESPONSE_FIELDS=               # Facultatif : liste blanche des champs de réponse de l'API (séparés par des virgules, par ex. phone,mail,user_id,status,name) ; vide = tous
export REMOTE_DECRYPT_ENABLED=false   # Facultatif : déchiffrer la réponse distante par RSA
export REMOTE_RSA_PRIVATE_KEY_FILE=   # Facultatif : chemin du PEM de clé privée RSA (ou REMOTE_RSA_PRIVATE_KEY pour un PEM en ligne)
export REMOTE_RSA_PRIVATE_KEY=        # Facultatif : PEM de clé privée RSA en ligne (utilisé si REMOTE_RSA_PRIVATE_KEY_FILE n'est pas défini)
export HTTP_TIMEOUT=5                  # Délai d'expiration des requêtes HTTP (secondes)
export HTTP_MAX_IDLE_CONNS=100         # Nombre maximal de connexions HTTP inactives
export HTTP_INSECURE_TLS=false         # Ignorer ou non la vérification des certificats TLS (true/false ou 1/0)
export API_KEY="your-secret-api-key"   # Clé d'API pour l'authentification (fortement recommandé)
export CONFIG_FILE=config.yaml         # Facultatif ; charge le traçage depuis le YAML sans `--config-file`, ou active le traçage depuis le même fichier que `--config-file`
export OTLP_ENABLED=false              # Activer OpenTelemetry (true/false ou 1/0)
export OTLP_ENDPOINT=http://localhost:4318  # Point de terminaison OTLP (obligatoire si OTLP_ENABLED est true)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # Liste des IP de proxy de confiance (séparées par des virgules)
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # Liste d'autorisation IP du point de terminaison de santé (facultatif)
export IP_WHITELIST="192.168.1.0/24"  # Liste d'autorisation IP globale (facultatif)
export LOG_LEVEL="info"                # Niveau de journalisation (facultatif, par défaut : info ; options : trace, debug, info, warn, error, fatal, panic)
export WARDEN_HMAC_KEYS='{"key-id":"0123456789abcdef0123456789abcdef"}'  # Les secrets de production exigent au moins 32 octets
export WARDEN_HMAC_ALLOW_V1=false                # Par défaut : false ; à passer à true uniquement pendant une migration v1 limitée dans le temps
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # Tolérance d'horodatage HMAC (secondes)
export WARDEN_TLS_CERT=/path/to/warden.crt    # Authentification de service : certificat TLS serveur (active TLS avec KEY)
export WARDEN_TLS_KEY=/path/to/warden.key     # Clé privée TLS du serveur
export WARDEN_TLS_CA=/path/to/ca.crt          # CA cliente (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # Exiger un certificat client (mTLS)
```

**Priorité des variables d'environnement** :
- Mot de passe Redis : `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > argument de ligne de commande `--redis-password`

**Remarques sur la configuration de sécurité** :
- `API_KEY` : protège les points de terminaison sensibles (`/`, `/log/level`) ; fortement recommandé en production
- `TRUSTED_PROXY_IPS` : configurez les IP des reverse proxys de confiance pour obtenir correctement l'IP réelle du client
- `HEALTH_CHECK_IP_WHITELIST` : restreint les IP d'accès au point de terminaison de santé (facultatif, prend en charge les plages CIDR)
- `IP_WHITELIST` : liste d'autorisation IP globale (facultatif, prend en charge les plages CIDR)

## Exigences de l'API de configuration distante

L'API de configuration distante doit renvoyer un tableau JSON au même format et peut éventuellement prendre en charge l'authentification par en-tête Authorization.

Le format de réponse de l'API doit correspondre à celui du fichier `data.json` :

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write"],
        "role": "admin"
    }
]
```

Si la variable d'environnement `KEY` ou le paramètre `--key` est configuré, l'en-tête `Authorization` est automatiquement ajouté aux requêtes :

```http
Authorization: Bearer your-token-here
```

## Configuration facultative de l'intégration de services

Si vous choisissez de vous intégrer à d'autres services (comme Stargate), l'authentification entre services peut être configurée. Voici les éléments de configuration concernés :

**Remarque** : si Warden est utilisé de manière autonome, les configurations suivantes sont facultatives.

### Configuration mTLS (recommandée)

Utiliser des certificats TLS mutuels pour l'authentification entre services. **Seules les variables d'environnement sont prises en charge** (aucune clé YAML dans la configuration de l'application) :

```bash
# Certificat serveur de Warden
export WARDEN_TLS_CERT=/path/to/warden.crt
export WARDEN_TLS_KEY=/path/to/warden.key
export WARDEN_TLS_CA=/path/to/ca.crt

# Exiger un certificat client (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
```

### Configuration de la signature HMAC

Utiliser une signature HMAC-SHA256 pour l'authentification entre services. **Seules les variables d'environnement sont prises en charge** (aucune clé YAML) :

```bash
# Clés HMAC (format JSON, plusieurs clés possibles pour la rotation)
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef","key-id-2":"abcdef0123456789abcdef0123456789"}'

# Tolérance d'horodatage (secondes), 60 par défaut lorsque des clés HMAC sont définies
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60

# L'ancien v1 est désactivé par défaut. Ne l'activez que pendant la migration des anciens appelants.
export WARDEN_HMAC_ALLOW_V1=false
```

### Configuration des appels depuis Stargate

Stargate doit configurer l'adresse du service Warden et les informations d'authentification :

**Exemple de configuration Stargate** (variables d'environnement) :
```bash
# Adresse du service Warden
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# Méthode d'authentification entre services (mTLS ou HMAC)
export STARGATE_WARDEN_AUTH_TYPE=hmac

# Configuration HMAC (si HMAC est utilisé)
export STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
export STARGATE_WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef

# Configuration mTLS (si mTLS est utilisé)
export STARGATE_WARDEN_TLS_CERT=/path/to/stargate.crt
export STARGATE_WARDEN_TLS_KEY=/path/to/stargate.key
export STARGATE_WARDEN_TLS_CA=/path/to/ca.crt
```

### Priorité de configuration

1. **mTLS** : si des certificats TLS sont configurés, mTLS est utilisé en premier
2. **HMAC** : si mTLS n'est pas configuré, la signature HMAC est utilisée
3. **Clé d'API** : si aucun des deux n'est configuré, repli sur l'authentification par clé d'API (déconseillé pour les appels entre services)

### Validation de la configuration

Au démarrage, Warden vérifie la configuration de l'authentification entre services :

- Rejette une configuration TLS partielle : le certificat et la clé doivent être définis ensemble ; mTLS exige en outre une CA cliente
- Si HMAC est configuré, vérifie que le format des clés est correct
- Sous `ENVIRONMENT=production`, refuse de démarrer si aucune authentification par clé d'API, HMAC ou mTLS n'est configurée

## Documentation détaillée de la configuration

Pour plus de détails sur les mécanismes d'analyse des paramètres, les règles de priorité et des exemples d'utilisation, consultez :

- [Document de conception de l'analyse des paramètres](CONFIG_PARSING.md) – Documentation détaillée du mécanisme d'analyse des paramètres
- [Document de conception de l'architecture](ARCHITECTURE.md) – Comprendre l'architecture globale et l'impact de la configuration
- [Documentation de sécurité](SECURITY.md) – Détails de l'authentification entre services
