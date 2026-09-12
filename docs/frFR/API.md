# Documentation de l'API

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](API.md) | [Italiano](../itIT/API.md) | [日本語](../jaJP/API.md) | [Deutsch](../deDE/API.md) | [한국어](../koKR/API.md)

Ce document fournit des informations détaillées sur tous les points de terminaison d'API proposés par Warden.

## Documentation OpenAPI

Le projet fournit une spécification OpenAPI 3.0 complète dans le fichier `openapi.yaml`.

Vous pouvez utiliser les outils suivants pour consulter et tester l'API :

1. **Swagger UI** : ouvrez le fichier `openapi.yaml` avec [Swagger Editor](https://editor.swagger.io/)
2. **Postman** : importez le fichier `openapi.yaml` dans Postman
3. **Redoc** : utilisez Redoc pour générer une belle page de documentation d'API

## Authentification

Certains points de terminaison exigent une authentification par clé d'API. Vous pouvez fournir les informations d'authentification de deux manières :

1. **En-tête X-API-Key** :
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **En-tête Authorization Bearer** :
   ```http
   Authorization: Bearer your-secret-api-key
   ```

La clé d'API se configure via la variable d'environnement `API_KEY` ou l'argument de ligne de commande `--api-key`.

## Contrat de routage

Les points de terminaison documentés ci-dessous constituent l'ensemble complet des chemins
servis par Warden. Tout autre chemin renvoie `404 Not Found` avec un corps JSON et sans
aucune donnée utilisateur :

```http
GET /not-a-route
X-API-Key: your-secret-api-key
```

```json
{
  "error": "Requested resource does not exist"
}
```

> **Changement de comportement** : le chemin racine `/` était auparavant enregistré comme
> motif de sous-arbre, si bien que tout chemin non apparié (`/foo`, `/user/`, `/v1/`) était
> servi par le gestionnaire de la liste des utilisateurs et renvoyait la **liste
> d'autorisation complète**. `/` est désormais une correspondance exacte et les chemins non
> appariés reçoivent le 404 ci-dessus. Les clients qui comptaient sur un chemin arbitraire
> pour obtenir des données utilisateur doivent utiliser `/`, `/data.json` ou `/v1/users`.

Notez que le routeur de Go nettoie le chemin avant l'appariement : `/metrics/../user` est
donc résolu en `/user` et redirigé vers celui-ci plutôt que d'atteindre le gestionnaire 404.

## Points de terminaison de l'API

### Obtenir la liste des utilisateurs

Obtenir tous les utilisateurs ou une liste paginée.

**Requête**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**Paramètres de requête** :
- `page` (facultatif) : numéro de page, à partir de 1, valeur par défaut 1
- `page_size` (facultatif) : nombre d'éléments par page, par défaut toutes les données (pas de pagination)

**Remarque** : ce point de terminaison exige une authentification par clé d'API.

**Réponse (sans pagination)**
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
```

**Réponse (avec pagination)**
```json
{
    "data": [
        {
            "phone": "13800138000",
            "mail": "admin@example.com"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 100,
        "total": 200,
        "total_pages": 2
    }
}
```

**Code de statut** : `200 OK`

**Content-Type** : `application/json`

### Obtenir un utilisateur unique

Interroger un utilisateur unique par numéro de téléphone, adresse e-mail ou identifiant d'utilisateur.

**Requête**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**Paramètres de requête** (exactement un doit être fourni) :
- `phone` : numéro de téléphone de l'utilisateur
- `mail` : adresse e-mail de l'utilisateur
- `user_id` : identifiant unique de l'utilisateur

**Remarque** :
- Ce point de terminaison exige une authentification par clé d'API
- Un seul paramètre de requête (`phone`, `mail` ou `user_id`) est autorisé

**Réponse (l'utilisateur existe)**
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**Description des champs** :
- `phone` : numéro de téléphone de l'utilisateur
- `mail` : adresse e-mail de l'utilisateur
- `user_id` : identifiant unique de l'utilisateur (généré automatiquement s'il n'est pas fourni)
- `status` : statut de l'utilisateur, valeurs possibles :
  - `"active"` : actif, l'utilisateur peut se connecter et accéder au système
  - `"inactive"` : inactif, l'utilisateur ne peut pas se connecter
  - `"suspended"` : suspendu, l'utilisateur ne peut pas se connecter
  - La valeur par défaut est `"inactive"` si rien n'est défini ; `"active"` doit être défini explicitement pour autoriser la connexion
- `scope` : tableau des portées de permission de l'utilisateur (facultatif), pour une autorisation fine, par exemple `["read", "write", "admin"]`
- `role` : rôle de l'utilisateur (facultatif), par exemple `"admin"`, `"user"`, `"guest"`

**Remarques** :
- Seuls les utilisateurs dont le `status` vaut `"active"` passent les contrôles d'authentification
- Les champs `scope` et `role` sont utilisés par Stargate pour définir les en-têtes d'autorisation (`X-Auth-Scopes` et `X-Auth-Role`) destinés aux services en aval

**Scénario d'intégration facultatif** :
Si vous choisissez de vous intégrer à d'autres services (comme Stargate), vous pouvez appeler ce point de terminaison pour interroger les informations utilisateur dans le flux de connexion :
1. Après que l'utilisateur a saisi un identifiant (e-mail/téléphone/nom d'utilisateur), appelez `GET /user?phone=xxx` ou `GET /user?mail=xxx`
2. Warden renvoie les informations utilisateur (y compris `user_id`, `mail`, `phone`, `status`)
3. Si l'utilisateur existe et que son statut est `"active"`, vous pouvez poursuivre le flux d'authentification
4. Les valeurs `scope` et `role` renvoyées peuvent servir à définir les en-têtes d'autorisation

**Réponse (utilisateur introuvable)**
- **Code de statut** : `404 Not Found`
- **Corps de la réponse** : `User not found`

**Réponse d'erreur (paramètre manquant)**
- **Code de statut** : `400 Bad Request`
- **Corps de la réponse** : `Bad Request: missing identifier (phone, mail, or user_id)`

**Réponse d'erreur (paramètres multiples)**
- **Code de statut** : `400 Bad Request`
- **Corps de la réponse** : `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### Contrôle de santé

Vérifie Redis, le cache de données, la provenance de l'instantané et sa fraîcheur.

**Requête**
```http
GET /health
GET /healthcheck
```

**Remarque** : ce point de terminaison n'exige pas d'authentification, mais les adresses IP autorisées peuvent être restreintes via la variable d'environnement `HEALTH_CHECK_IP_WHITELIST`. En production, les réponses masquent les contrôles individuels.

**Réponse**
```json
{
    "status": "ok",
    "service": "warden",
    "checks": {
        "redis": {
            "name": "redis",
            "status": "ok",
            "latency_ms": 1,
            "timestamp": "2026-08-31T00:00:00Z"
        },
        "snapshot": {
            "name": "snapshot",
            "status": "ok",
            "latency_ms": 0,
            "timestamp": "2026-08-31T00:00:00Z",
            "metadata": {
                "source": "merged",
                "version": "a1b2c3d4",
                "age_seconds": 2.5
            }
        }
    },
    "timestamp": "2026-08-31T00:00:00Z",
    "total_latency_ms": 1
}
```

Réponse en production :

```json
{"status":"ok","service":"warden"}
```

**Codes de statut** :

- `200 OK` : le statut global est `ok` ou `degraded` ; `degraded` signifie que le service reste fonctionnel.
- `503 Service Unavailable` : un contrôle critique a échoué.
- `403 Forbidden` : le client est en dehors de `HEALTH_CHECK_IP_WHITELIST`.

**Description des champs de réponse** :
- `status` : `ok`, `degraded` ou `unhealthy`.
- `service` : nom du service (`warden`).
- `checks` : table présente uniquement en développement/test, contenant les résultats de `redis`, `data`, `snapshot` et `snapshot_freshness`.
- `checks.snapshot.metadata` : source, version et âge à faible cardinalité, plus des codes de raison stables pour les rafraîchissements ; les erreurs distantes brutes, les URL et les identifiants ne sont jamais exposés.
- `timestamp`, `total_latency_ms` : champs de chronométrage global présents uniquement en développement/test.

En `REMOTE_FIRST` et `ONLY_REMOTE`, `snapshot_freshness` est critique. Une provenance
inconnue ou un âge supérieur à `SNAPSHOT_MAX_AGE` renvoie 503. Les modes tolérants peuvent
servir un instantané local validé ou le dernier instantané valide connu en état `degraded`
avec un HTTP 200.

### Gestion du niveau de journalisation

Obtenir et définir dynamiquement les niveaux de journalisation.

#### Obtenir le niveau de journalisation actuel

**Requête**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**Réponse**
```json
{
    "level": "info"
}
```

**Remarque** : ce point de terminaison exige une authentification par clé d'API.

#### Définir le niveau de journalisation

**Requête**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**Corps de la requête** :
```json
{
    "level": "debug"
}
```

**Niveaux de journalisation pris en charge** : `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**Réponse**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**Remarque** :
- Ce point de terminaison exige une authentification par clé d'API
- Toutes les opérations de modification du niveau de journalisation sont enregistrées dans les journaux d'audit de sécurité

### Métriques Prometheus

Obtenir les données de métriques de supervision au format Prometheus.

**Requête**
```http
GET /metrics
```

**Réponse** : données de métriques au format Prometheus

**Authentification** : dépend de l'environnement de déploiement.

| `ENVIRONMENT` | Valeur par défaut pour `/metrics` |
| --- | --- |
| `production` | Authentification requise (mêmes schémas que pour les points de terminaison de données) |
| `development`, `test`, non défini | Collecte anonyme autorisée |

`WARDEN_METRICS_REQUIRE_AUTH` remplace la valeur par défaut dans les deux sens. Une collecte
non authentifiée d'un point de terminaison qui exige une authentification renvoie
`401 Unauthorized`. La réponse ne contient que des séries à faible cardinalité et non
sensibles : les étiquettes `endpoint` et `method` sont normalisées selon une liste
d'autorisation et les valeurs non reconnues sont regroupées sous `other`.

**Exemple de réponse** :
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 1234

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/",le="0.005"} 1000
http_request_duration_seconds_bucket{method="GET",path="/",le="0.01"} 1200
...
```

## Réponses d'erreur

Tous les points de terminaison peuvent renvoyer les réponses d'erreur suivantes :

### 401 Unauthorized

Renvoyée lorsque l'authentification par clé d'API échoue :

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

Renvoyée lorsque les requêtes dépassent la limite de débit :

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

Renvoyée lorsqu'une erreur interne du serveur se produit :

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

En mode production, les informations d'erreur détaillées sont masquées afin d'éviter toute fuite d'informations.

## Limitation de débit

Par défaut, les requêtes d'API sont protégées par une limitation de débit :

- **Limite** : 60 requêtes par minute
- **Fenêtre** : 1 minute
- **En cas de dépassement** : renvoie `429 Too Many Requests`

La limitation de débit peut être ajustée via le fichier de configuration :

```yaml
rate_limit:
  rate: 60  # Requêtes par minute
  window: 1m
```

## Liste d'autorisation d'adresses IP

Les listes d'autorisation d'adresses IP se configurent via les variables d'environnement suivantes :

- `IP_WHITELIST` : liste d'autorisation globale (restreint l'accès à tous les points de terminaison)
- `HEALTH_CHECK_IP_WHITELIST` : liste d'autorisation pour le point de terminaison de contrôle de santé (restreint uniquement `/health` et `/healthcheck`)

Le format de plage CIDR est pris en charge ; plusieurs adresses IP ou plages sont séparées par des virgules :

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Compression des réponses

Toutes les réponses de l'API prennent en charge la compression automatique (gzip). Les clients peuvent activer la compression via l'en-tête de requête `Accept-Encoding: gzip`.

## Exemples d'intégration facultatifs

### Exemple d'appel pour l'intégration avec d'autres services (facultatif)

Si vous devez vous intégrer à d'autres services (comme Stargate), vous pouvez appeler le point de terminaison `/user` de Warden pour interroger les informations utilisateur dans le flux de connexion :

**Scénario 1 : interrogation par numéro de téléphone**

```bash
# Stargate appelle Warden
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?phone=13800138000"
```

**Exemple de réponse** :
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**Scénario 2 : interrogation par e-mail**

```bash
# Stargate appelle Warden
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?mail=admin@example.com"
```

### Exemple d'intégration avec le SDK Go

Stargate peut utiliser le SDK Go de Warden pour l'intégration :

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Créer le client Warden
    opts := warden.DefaultOptions().
        WithBaseURL("http://warden:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second)
    
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // Interroger l'utilisateur dans le flux de connexion
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            // Utilisateur introuvable, refuser la connexion
            fmt.Println("User not found in allowlist")
            return
        }
        panic(err)
    }
    
    // Vérifier le statut de l'utilisateur
    if !user.IsActive() {
        // Le statut de l'utilisateur n'est pas actif, refuser la connexion
        fmt.Printf("User status is %s, cannot login\n", user.Status)
        return
    }
    
    // L'utilisateur existe et son statut est actif, poursuivre le flux de connexion
    fmt.Printf("User found: %s, Status: %s, Role: %s, Scopes: %v\n",
        user.UserID, user.Status, user.Role, user.Scope)
    
    // Étape suivante : appeler Herald pour envoyer un code de vérification
    // ...
}
```

### Exemple de flux de connexion complet (scénario d'intégration facultatif)

Dans les scénarios d'intégration facultatifs, le flux de connexion complet peut se présenter ainsi :

1. **L'utilisateur saisit un identifiant** → le service d'authentification le reçoit
2. **Service d'authentification → Warden** : interroger les informations utilisateur
   ```go
   user, err := wardenClient.GetUserByIdentifier(ctx, phone, mail, "")
   ```
3. **Valider le statut de l'utilisateur** : vérifier `user.Status == "active"`
4. **Service d'authentification → service OTP** : créer un défi et envoyer le code de vérification (facultatif)
5. **L'utilisateur soumet le code de vérification** → le service d'authentification le reçoit (facultatif)
6. **Service d'authentification → service OTP** : vérifier le code (facultatif)
7. **Service d'authentification** : émettre une session et utiliser `user.Scope` et `user.Role` pour définir les en-têtes d'autorisation

**Remarque** : Warden peut être utilisé de manière autonome ; le flux d'intégration ci-dessus est facultatif.

## Documentation associée

- [Spécification OpenAPI](../../openapi.yaml) – Spécification OpenAPI 3.1 complète
- [Documentation de configuration](CONFIGURATION.md) – Découvrez comment configurer la clé d'API et les autres options
- [Documentation de sécurité](SECURITY.md) – Découvrez les fonctionnalités de sécurité et les bonnes pratiques
- [Documentation d'architecture](ARCHITECTURE.md) – Découvrez l'architecture d'intégration des services
