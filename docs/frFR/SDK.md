# Documentation d'utilisation du SDK

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](../zhCN/SDK.md) | [Français](SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](../deDE/SDK.md) | [한국어](../koKR/SDK.md)

Warden fournit un SDK Go pour faciliter l'intégration dans d'autres projets. Le SDK offre une interface d'API épurée avec prise en charge du cache, de l'authentification, etc.

## Fonctionnalités

- 🚀 **Simple et accessible** : fournit des interfaces d'API épurées
- ⚡ **Performant** : cache intégré (GetUsers) ; les requêtes directes (GetUserByIdentifier) réduisent les appels à l'API
- 🔒 **Sûr** : prend en charge l'authentification par clé d'API ; la gestion des erreurs ne divulgue aucune information sensible
- 📦 **Flexible** : délai d'expiration, TTL du cache, etc. configurables
- 🔌 **Extensible** : prend en charge des implémentations de logger personnalisées
- 🎯 **Repli intelligent** : CheckUserInList bascule automatiquement sur l'e-mail lorsque le téléphone n'est pas trouvé

## Installer le SDK

```bash
go get github.com/soulteary/warden/pkg/warden
```

## Démarrage rapide

### Utilisation de base

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Créer les options du client
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)
    
    // Créer le client
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    // Récupérer la liste des utilisateurs
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }
    
    // Vérifier si l'utilisateur est dans la liste (phone, mail ou les deux)
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // On peut aussi n'utiliser que phone ou que mail
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // Récupérer les détails de l'utilisateur
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### Utiliser un logger personnalisé

Le SDK prend en charge des implémentations de logger personnalisées, par exemple avec logrus :

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    logger := logrus.StandardLogger()
    
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithLogger(warden.NewLogrusAdapter(logger))
    
    client, err := warden.NewClient(opts)
    // ...
}
```

### Requête paginée

```go
// Récupérer la liste paginée des utilisateurs
resp, err := client.GetUsersPaginated(ctx, 1, 10) // Page 1, 10 éléments par page
if err != nil {
    panic(err)
}

fmt.Printf("Total users: %d\n", resp.Pagination.Total)
fmt.Printf("Total pages: %d\n", resp.Pagination.TotalPages)
for _, user := range resp.Data {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
}
```

### Récupérer les informations d'un utilisateur unique

```go
// Récupérer les informations de l'utilisateur par téléphone
user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
        println("User not found")
    } else {
        panic(err)
    }
} else {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
    if user.IsActive() {
        println("User is active")
    }
}

// Récupérer les informations de l'utilisateur par e-mail
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// Récupérer les informations de l'utilisateur par identifiant
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
```

### Vider le cache

```go
// Vider manuellement le cache du client
client.ClearCache()

// Ou utiliser l'alias
client.InvalidateCache()
```

### Transport HTTP personnalisé

```go
import "net/http"

// Créer un transport personnalisé
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### Signature de requête HMAC v2

```go
opts := warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET"))

client, err := warden.NewClient(opts)
```

Le SDK signe l'identifiant de clé, l'horodatage, le nonce, le chemin/la requête, la méthode et l'empreinte du corps.
L'identifiant de clé et le secret doivent tous deux être configurés ; une paire incomplète renvoie
`ErrCodeInvalidConfig` au lieu d'envoyer une requête non signée.

### Configuration des nouvelles tentatives

```go
// Configurer les options de nouvelle tentative
retryOpts := warden.DefaultRetryOptions()
retryOpts.MaxRetries = 3
retryOpts.RetryDelay = 100 * time.Millisecond
retryOpts.MaxRetryDelay = 5 * time.Second
retryOpts.BackoffMultiplier = 2.0

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithRetry(retryOpts)

client, err := warden.NewClient(opts)
```

### Invalidation du cache pilotée par événement

```go
// Créer un canal pour les événements d'invalidation du cache
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // Important : fermer pour arrêter l'écouteur en arrière-plan

// Plus tard, déclencher l'invalidation du cache depuis un événement externe
invalidationCh <- struct{}{}

// Le cache sera automatiquement vidé à la réception du signal
```

## Référence de l'API

### Options

La structure `Options` sert à configurer le client :

- `BaseURL` : adresse du service Warden (obligatoire)
- `APIKey` : clé d'API (facultatif)
- `Timeout` : délai d'expiration des requêtes HTTP (10 secondes par défaut)
- `CacheTTL` : TTL du cache (5 minutes par défaut)
- `Logger` : interface de logger (facultatif, NoOpLogger par défaut)
- `Transport` : transport HTTP personnalisé (facultatif)
- `HMACKeyID` / `HMACSecret` : paire de signature HMAC v2 (les deux ou aucun)
- `TLSConfig` : configuration client TLS/mTLS (facultatif)
- `Retry` : configuration des nouvelles tentatives (facultatif, aucune tentative par défaut)
- `CacheInvalidationChannel` : canal pour l'invalidation du cache pilotée par événement (facultatif)

### Méthodes du client

#### `NewClient(opts *Options) (*Client, error)`

Crée un nouveau client Warden.

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

Récupère la liste complète des utilisateurs. Si le cache est valide, les données mises en cache sont renvoyées directement.

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

Récupère la liste paginée des utilisateurs.

- `page` : numéro de page (à partir de 1)
- `pageSize` : taille de page

Renvoie `PaginatedResponse`, contenant :
- `Data` : liste des utilisateurs
- `Pagination` : informations de pagination (numéro de page, taille de page, total, nombre total de pages)

**Remarque :** cette méthode n'utilise pas le cache ; chaque appel récupère les données les plus récentes auprès de l'API.

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

Récupère les informations d'un utilisateur unique à partir d'un identifiant.

- `phone` : numéro de téléphone de l'utilisateur (facultatif, mais l'un de phone, mail ou userID doit être fourni)
- `mail` : e-mail de l'utilisateur (facultatif)
- `userID` : identifiant unique de l'utilisateur (facultatif)

**Important :** exactement un identifiant parmi `phone`, `mail` ou `userID` doit être fourni.

Renvoie `*AllowListUser` et une erreur. Si l'utilisateur n'existe pas, renvoie l'erreur `ErrCodeNotFound`.

**Remarque :** cette méthode n'utilise pas le cache ; chaque appel récupère les données les plus récentes auprès de l'API.

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

Vérifie si un utilisateur figure dans la liste d'autorisation.

- `phone` : numéro de téléphone de l'utilisateur (facultatif)
- `mail` : e-mail de l'utilisateur (facultatif)

Renvoie `true` si l'utilisateur existe (trouvé par téléphone ou e-mail), `false` sinon.

**Comportement :**
- Si `phone` et `mail` sont tous deux fournis, `phone` est prioritaire
- Si la recherche par `phone` échoue (erreur `NotFound`) et que `mail` n'est pas vide, bascule automatiquement sur la recherche par `mail`
- Si la recherche par `phone` réussit mais que le statut de l'utilisateur n'est pas actif, aucun repli sur `mail` (l'utilisateur a déjà été trouvé)
- Si la recherche par `phone` échoue avec une erreur autre que `NotFound` (par ex. une erreur réseau), aucun repli sur `mail`
- L'entrée est normalisée automatiquement : `phone` est débarrassé des espaces, `mail` est débarrassé des espaces et converti en minuscules
- Cette méthode utilise `GetUserByIdentifier` pour la recherche, ce qui est plus efficace que de parcourir la liste des utilisateurs
- Seuls les utilisateurs dont le statut est « active » renvoient `true`

#### `ClearCache()`

Vide le cache interne du client.

#### `InvalidateCache()`

Alias de `ClearCache()`, par cohérence avec l'invalidation pilotée par événement.

#### `Close()`

Arrête les goroutines d'arrière-plan (par ex. l'écouteur d'invalidation du cache) et libère les ressources.
À appeler lorsque le client n'est plus nécessaire.

## Définitions de types

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // Numéro de téléphone de l'utilisateur
    Mail   string   `json:"mail"`    // Adresse e-mail de l'utilisateur
    UserID string   `json:"user_id"` // Identifiant unique de l'utilisateur (facultatif, généré automatiquement)
    Status string   `json:"status"`  // Statut de l'utilisateur (par ex. "active", "inactive", "suspended")
    Scope  []string `json:"scope"`   // Portée de permission de l'utilisateur (facultatif)
    Role   string   `json:"role"`    // Rôle de l'utilisateur (facultatif)
}
```

**Méthodes :**
- `IsActive() bool` : vérifie si le statut de l'utilisateur est « active »
- `IsValid() bool` : vérifie si le statut de l'utilisateur est valide (seul « active » est pris en charge actuellement)

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // Numéro de page courant (à partir de 1)
    PageSize   int `json:"page_size"`   // Taille de page
    Total      int `json:"total"`       // Nombre total d'enregistrements
    TotalPages int `json:"total_pages"` // Nombre total de pages
}
```

## Gestion des erreurs

Le SDK utilise des types d'erreur personnalisés comportant des codes d'erreur et des informations détaillées :

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // Gérer l'erreur d'authentification
        case warden.ErrCodeRequestFailed:
            // Gérer l'échec de la requête
        case warden.ErrCodeNotFound:
            // Gérer l'erreur « non trouvé »
        case warden.ErrCodeServerError:
            // Gérer l'erreur serveur
        // ...
        }
    }
}
```

### Codes d'erreur

- `ErrCodeInvalidConfig` : configuration invalide
- `ErrCodeRequestFailed` : échec de la requête
- `ErrCodeInvalidResponse` : format de réponse invalide
- `ErrCodeUnauthorized` : non autorisé
- `ErrCodeNotFound` : non trouvé
- `ErrCodeServerError` : erreur serveur

## Bonnes pratiques

1. **Réutiliser le client** : créez le client une fois et réutilisez-le pendant toute la durée de vie de l'application
2. **Choisir un TTL de cache adapté** : définissez une durée de cache adaptée à la fréquence de mise à jour des données
3. **Utiliser Context** : transmettez un contexte pour permettre l'annulation et le contrôle du délai d'expiration
4. **Gestion des erreurs** : vérifiez et traitez toujours les erreurs
5. **Journalisation** : utilisez une implémentation de logger appropriée en production
6. **Fermer le client** : appelez `Close()` lorsque le client n'est plus nécessaire, afin d'arrêter les goroutines d'arrière-plan
7. **Configurer les nouvelles tentatives** : activez-les en production pour absorber les défaillances transitoires
8. **Transport personnalisé** : utilisez un transport personnalisé pour les scénarios avancés (TLS, proxy, pool de connexions, etc.)

## Documentation de conception

### Principes de conception

1. **Simple et accessible** : fournit des interfaces d'API épurées
2. **Performant** : le cache intégré réduit les appels à l'API
3. **Sûr en concurrence** : toutes les méthodes sont utilisables de façon concurrente
4. **Configuration flexible** : délai d'expiration, cache, logger, etc. personnalisables

### Conception de l'architecture

#### Composants principaux

1. **Client** : encapsulation du client HTTP
2. **Cache** : cache en mémoire sûr en concurrence
3. **Options** : options de configuration (motif Builder)
4. **Logger** : interface de logger (prend en charge différentes bibliothèques de journalisation)

#### Sûreté en concurrence

- `http.Client` est sûr en concurrence
- `Cache` utilise `sync.RWMutex` pour garantir la sûreté vis-à-vis des threads
- Tous les champs de `Client` sont en lecture seule après création
- Toutes les méthodes sont sûres et peuvent être appelées de façon concurrente depuis plusieurs goroutines

#### Stratégie de cache

1. **GetUsers()** : utilise le cache
   - Consulte d'abord le cache
   - Si le cache est valide, renvoie directement
   - Si le cache est invalide ou absent, récupère les données depuis l'API et met à jour le cache

2. **GetUsersPaginated()** : n'utilise pas le cache
   - Raison : des paramètres de pagination différents produisent des résultats différents
   - Mettre en cache par paramètres de pagination serait complexe
   - Conception actuelle : récupère les données depuis l'API à chaque appel pour garantir leur exactitude

3. **GetUserByIdentifier()** : n'utilise pas le cache
   - Raison : il faut obtenir les informations les plus récentes d'un utilisateur unique pour garantir la fraîcheur des données
   - Chaque appel interroge l'API afin d'éviter les incohérences dues au cache

4. **CheckUserInList()** : n'utilise pas le cache
   - Utilise `GetUserByIdentifier()` pour interroger directement un utilisateur unique
   - Chaque appel envoie une requête à l'API afin de garantir la fraîcheur des données
   - Prend en charge un repli intelligent : lorsque la recherche par téléphone échoue (NotFound) et que mail n'est pas vide, bascule automatiquement sur mail
   - Optimisation des performances : interroger directement un utilisateur unique est plus efficace que parcourir toute la liste des utilisateurs

#### Stratégie d'implémentation de CheckUserInList

La méthode `CheckUserInList()` applique la stratégie suivante :

1. **Normalisation de l'entrée** : supprime automatiquement les espaces en début et fin de phone et de mail, et convertit mail en minuscules
2. **Stratégie de priorité** : si phone et mail sont tous deux fournis, phone est prioritaire
3. **Repli intelligent** :
   - Lorsque la recherche par phone renvoie l'erreur `NotFound`, si mail n'est pas vide, bascule automatiquement sur la recherche par mail
   - Lorsque la recherche par phone réussit mais que le statut de l'utilisateur n'est pas actif, aucun repli sur mail (l'utilisateur a déjà été trouvé)
   - Lorsque la recherche par phone rencontre une autre erreur (par ex. une erreur réseau), aucun repli sur mail
4. **Validation du statut** : seuls les utilisateurs dont le statut est « active » renvoient `true`
5. **Optimisation des performances** : utilise `GetUserByIdentifier()` pour une requête directe, évitant de récupérer toute la liste des utilisateurs

### RetryOptions

La structure `RetryOptions` configure le comportement des nouvelles tentatives :

- `MaxRetries` : nombre maximal de tentatives (0 par défaut, aucune tentative)
- `RetryDelay` : délai initial entre les tentatives (100 ms par défaut)
- `MaxRetryDelay` : délai maximal entre les tentatives (5 s par défaut)
- `BackoffMultiplier` : multiplicateur du backoff exponentiel (2.0 par défaut)
- `RetryableStatusCodes` : codes de statut HTTP déclenchant une nouvelle tentative (par défaut : 5xx)

**Remarque :** les erreurs réseau sont toujours réessayables. Les erreurs client (4xx) ne le sont jamais.

### Limites connues

1. **Cache de pagination** : `GetUsersPaginated()` n'utilise pas le cache
   - Il s'agit d'un choix délibéré pour garantir l'exactitude des données
   - Si un cache de pagination est nécessaire, des stratégies plus complexes peuvent être mises en œuvre

2. **Cache des requêtes d'utilisateur unique** : `GetUserByIdentifier()` et `CheckUserInList()` n'utilisent pas le cache
   - Il s'agit d'un choix délibéré pour garantir la fraîcheur des données
   - Si un cache est nécessaire, des stratégies fondées sur les identifiants d'utilisateur peuvent être mises en œuvre

### Améliorations futures

1. Prise en charge de middlewares requête/réponse
2. Prise en charge de la collecte de métriques
3. Prise en charge de la configuration du pool de connexions
4. Prise en charge du motif disjoncteur (circuit breaker)

## Exemple complet

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Créer le client
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)

    client, err := warden.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Récupérer tous les utilisateurs
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // Récupérer un utilisateur unique par téléphone
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            fmt.Println("User not found")
        } else {
            log.Fatal(err)
        }
    } else {
        fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
    }

    // Requête paginée
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // Vérifier l'utilisateur
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // Vider le cache
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## Documentation associée

- [Documentation de l'API](API.md) – Découvrez le détail des points de terminaison de l'API
- [Documentation de configuration](CONFIGURATION.md) – Découvrez les options de configuration du serveur
