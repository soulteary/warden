# Warden OpenTelemetry Tracing

Le service Warden prend en charge le traçage distribué OpenTelemetry pour surveiller et déboguer les chaînes d'appels inter-services.

## Fonctionnalités

- **Traçage automatique des requêtes HTTP** : Crée automatiquement des spans pour toutes les requêtes HTTP
- **Traçage des requêtes utilisateur** : Ajoute des informations de traçage détaillées pour le point de terminaison `/user`
- **Propagation du contexte** : Prend en charge la norme W3C Trace Context, s'intègre parfaitement avec les services Stargate et Herald
- **Configurable** : Activer/désactiver via des variables d'environnement ou des fichiers de configuration

## Configuration

### Variables d'environnement

```bash
# Activer le traçage OpenTelemetry
OTLP_ENABLED=true

# Point de terminaison OTLP (par exemple : Jaeger, Tempo, OpenTelemetry Collector)
OTLP_ENDPOINT=http://localhost:4318
```

### Fichier de configuration (YAML)

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

## Spans principaux

### Span de requête HTTP

Toutes les requêtes HTTP créent automatiquement des spans avec les attributs suivants :
- `http.method`: Méthode HTTP
- `http.url`: URL de la requête
- `http.status_code`: Code de statut de la réponse
- `http.user_agent`: Agent utilisateur
- `http.remote_addr`: Adresse du client

### Span de requête utilisateur (`warden.get_user`)

Les requêtes au point de terminaison `/user` créent des spans dédiés contenant :
- `warden.query.phone`: Numéro de téléphone interrogé (masqué)
- `warden.query.mail`: Email interrogé (masqué)
- `warden.query.user_id`: ID utilisateur interrogé
- `warden.user.found`: Si l'utilisateur a été trouvé
- `warden.user.id`: ID utilisateur trouvé

## Exemples d'utilisation

### Démarrer Warden avec le traçage activé

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### Utiliser le traçage dans le code

```go
import "github.com/soulteary/warden/internal/tracing"

// Créer un span enfant
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// Définir des attributs
span.SetAttributes(attribute.String("key", "value"))

// Enregistrer une erreur
if err != nil {
    tracing.RecordError(span, err)
}
```

## Intégration avec Stargate et Herald

Le traçage de Warden s'intègre automatiquement avec le contexte de traçage des services Stargate et Herald :

1. **Stargate** transmet le contexte de trace via les en-têtes HTTP lors de l'appel à Warden
2. **Warden** extrait automatiquement et continue la chaîne de traçage
3. Les spans des trois services apparaissent dans la même trace

## Backends de traçage pris en charge

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **Autres backends compatibles OTLP**

## Considérations de performance

- Le traçage utilise l'export par lots par défaut, minimisant l'impact sur les performances
- Le volume de données de trace peut être contrôlé via le taux d'échantillonnage
- Les environnements de production devraient utiliser des stratégies d'échantillonnage (actuellement échantillonnage complet, adapté au développement)

## Dépannage

### Traçage non activé

Vérifier les variables d'environnement :
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### Les données de trace n'atteignent pas le backend

1. Vérifier si le point de terminaison OTLP est accessible
2. Vérifier la connexion réseau
3. Consulter les messages d'erreur dans les journaux Warden

### Spans manquants

Assurez-vous d'utiliser `r.Context()` pour transmettre le contexte dans le traitement des requêtes, plutôt que de créer un nouveau contexte.
