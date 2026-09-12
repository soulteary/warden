# Documentation de déploiement

> 🌐 **Language / 语言**: [English](../enUS/DEPLOYMENT.md) | [中文](../zhCN/DEPLOYMENT.md) | [Français](DEPLOYMENT.md) | [Italiano](../itIT/DEPLOYMENT.md) | [日本語](../jaJP/DEPLOYMENT.md) | [Deutsch](../deDE/DEPLOYMENT.md) | [한국어](../koKR/DEPLOYMENT.md)

Ce document explique comment déployer le service Warden : déploiement Docker, déploiement local, etc.

## Prérequis

- Go 1.27+ (voir [go.mod](../../go.mod))
- Redis (pour les verrous distribués et la mise en cache)
- Docker (facultatif, pour un déploiement conteneurisé)

## Déploiement Docker

> 🚀 **Déploiement rapide** : consultez le [répertoire d'exemples](../../example/README.md) / [示例目录](../../example/README.md) pour des exemples complets de configuration Docker Compose :
> - [Exemple simple](../../example/basic/docker-compose.yml) / [简单示例](../../example/basic/docker-compose.yml) – Configuration Docker Compose de base
> - [Exemple avancé](../../example/advanced/docker-compose.yml) / [复杂示例](../../example/advanced/docker-compose.yml) – Configuration complète incluant une API fictive

### Utiliser l'image pré-construite (recommandé)

Warden fournit des images Docker pré-construites, récupérables directement depuis GitHub Container Registry (GHCR), sans build manuel :

```bash
# Récupérer l'image de la dernière version
docker pull ghcr.io/soulteary/warden:latest

# Lancer le conteneur
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api/data.json \
  -e KEY="Bearer your-token-here" \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Astuce** : les images pré-construites permettent de démarrer rapidement sans environnement de build local. Elles sont mises à jour automatiquement pour vous garantir la dernière version.

### Utiliser Docker Compose

1. **Préparer le fichier de variables d'environnement**
   
   Si un fichier `.env.example` existe à la racine du projet, vous pouvez le copier :
   ```bash
   cp .env.example .env
   ```
   
   Si le fichier `.env.example` n'existe pas, vous pouvez créer manuellement un fichier `.env` avec le contenu suivant :
   ```env
   # Configuration du serveur
   PORT=8081
   
   # Configuration Redis
   REDIS=warden-redis:6379
   # Mot de passe Redis (facultatif, préférez les variables d'environnement au fichier de configuration)
   # REDIS_PASSWORD=your-redis-password
   # Ou utiliser un fichier de mot de passe (plus sûr)
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # API de données distante
   CONFIG=http://example.com/api/data.json
   # Clé d'authentification de l'API de configuration distante
   KEY=Bearer your-token-here
   
   # Configuration des tâches
   INTERVAL=5
   
   # Mode de l'application
   MERGE_MODE=DEFAULT
   
   # Configuration du client HTTP (facultatif)
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # Clé d'API (pour l'authentification de l'API, obligatoire en production)
   API_KEY=your-api-key-here
   
   # Liste d'autorisation IP du contrôle de santé (facultatif, séparée par des virgules)
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # Liste des IP de proxy de confiance (facultatif, séparée par des virgules, pour les environnements avec reverse proxy)
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # Niveau de journalisation (facultatif)
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **Note de sécurité** : le fichier `.env` contient des informations sensibles. Ne le committez pas dans le système de gestion de versions. Le fichier `.env` est déjà ignoré par `.gitignore`. Utilisez le contenu ci-dessus comme modèle pour créer votre fichier `.env`.

2. **Démarrer le service**
```bash
docker-compose up -d
```

### Construction manuelle de l'image

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### Lancer le conteneur

```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api \
  -e KEY="Bearer token" \
  warden-release
```

## Déploiement local

### 1. Cloner le projet

```bash
git clone <repository-url>
cd warden
```

### 2. Installer les dépendances

```bash
go mod download
```

### 3. Configurer le fichier de données local

Créez un fichier `data.json` (voir `data.example.json`) :
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Remarque** : le fichier `data.json` prend en charge les champs suivants :
- `phone` (obligatoire) : numéro de téléphone de l'utilisateur
- `mail` (obligatoire) : adresse e-mail de l'utilisateur
- `user_id` (facultatif) : identifiant unique de l'utilisateur, généré automatiquement s'il n'est pas fourni
- `status` (facultatif) : statut de l'utilisateur, par exemple « active », « inactive », « suspended » ; en l'absence de valeur, le statut par défaut est « inactive »
- `scope` (facultatif) : tableau des portées de permission de l'utilisateur, par exemple `["read", "write"]`
- `role` (facultatif) : rôle de l'utilisateur, par exemple « admin », « user »

Pour un exemple complet, consultez le fichier `data.example.json`.

### 4. Lancer le service

```bash
go run .
```

## Recommandations de déploiement en production

### 1. Utiliser un reverse proxy

En production, il est recommandé d'utiliser un reverse proxy tel que Nginx ou Traefik :

**Exemple de configuration Nginx** :
```nginx
upstream warden {
    server localhost:8081;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://warden;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. Utiliser HTTPS

Les environnements de production doivent utiliser HTTPS. Vous pouvez y parvenir en :

- Utilisant les certificats gratuits de Let's Encrypt
- Utilisant un reverse proxy (comme Nginx) pour gérer SSL/TLS
- Configurant la variable d'environnement `TRUSTED_PROXY_IPS` afin d'obtenir correctement l'IP réelle du client

### 3. Configurer la supervision

- Utiliser Prometheus pour collecter les métriques (via le point de terminaison `/metrics`)
- Configurer les contrôles de santé (via le point de terminaison `/health`)
- Mettre en place la collecte et l'analyse des journaux

### 4. Déploiement en haute disponibilité

- Déployer plusieurs instances et répartir les requêtes avec un équilibreur de charge
- Utiliser une instance Redis partagée pour garantir la cohérence des données
- Configurer le redémarrage automatique et le basculement

### 5. Limites de ressources

Configurez les limites de ressources dans Docker Compose ou Kubernetes :

```yaml
services:
  warden:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Déploiement Kubernetes

### Déploiement de base

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "redis-service:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: warden-service
spec:
  selector:
    app: warden
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  type: LoadBalancer
```

## Optimisation des performances

### 1. Configuration Redis

- Utiliser la persistance Redis (RDB ou AOF)
- Configurer des limites de mémoire Redis appropriées
- Utiliser un cluster Redis (si nécessaire)

### 2. Configuration de l'application

- Ajuster `HTTP_MAX_IDLE_CONNS` pour optimiser le pool de connexions
- Configurer un `INTERVAL` approprié pour équilibrer réactivité et efficacité
- Utiliser un mode de fusion (`MERGE_MODE`) adapté

### 3. Supervision et réglage

D'après les résultats d'un test de charge wrk (test de 30 secondes, 16 threads, 100 connexions) :

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

Ajustez les paramètres de configuration en fonction de la charge réelle.

## Déploiement avec intégration facultative (avec Stargate/Herald)

Warden peut être déployé et utilisé seul, ou éventuellement intégré à Stargate et Herald. Voici des exemples de configuration pour un déploiement intégré facultatif.

**Remarque** : les scénarios de déploiement intégré ci-dessous sont facultatifs ; Warden peut être déployé et utilisé de manière totalement indépendante.

### Exemple d'intégration avec Docker Compose

Configuration complète de déploiement intégrant Stargate + Warden + Herald :

```yaml
version: '3.8'

services:
  # Service Warden
  warden:
    image: ghcr.io/soulteary/warden:latest
    container_name: warden
    ports:
      - "8081:8081"
    networks:
      - auth-network
    environment:
      - PORT=8081
      - REDIS=warden-redis:6379
      - API_KEY=${WARDEN_API_KEY}
      - MERGE_MODE=DEFAULT
      # Configuration de l'authentification entre services (exemple HMAC)
      - WARDEN_HMAC_KEYS=${WARDEN_HMAC_KEYS}
      - WARDEN_HMAC_TIMESTAMP_TOLERANCE=60
    volumes:
      - ./warden-data.json:/app/data.json:ro
    healthcheck:
      test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
      interval: 10s
      timeout: 1s
      retries: 3
    depends_on:
      - warden-redis

  # Redis de Warden
  warden-redis:
    image: redis:7.4-alpine
    container_name: warden-redis
    networks:
      - auth-network
    volumes:
      - warden-redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 1s
      retries: 3

  # Service Stargate (exemple de configuration)
  stargate:
    image: ghcr.io/soulteary/stargate:latest
    container_name: stargate
    ports:
      - "8080:8080"
    networks:
      - auth-network
    environment:
      - STARGATE_WARDEN_BASE_URL=http://warden:8081
      - STARGATE_WARDEN_AUTH_TYPE=hmac
      - STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
      - STARGATE_WARDEN_HMAC_SECRET=${WARDEN_HMAC_SECRET}
      - STARGATE_HERALD_BASE_URL=http://herald:8082
    depends_on:
      - warden
      - herald

  # Service Herald (exemple de configuration)
  herald:
    image: ghcr.io/soulteary/herald:latest
    container_name: herald
    ports:
      - "8082:8082"
    networks:
      - auth-network
    environment:
      - HERALD_REDIS_URL=redis://herald-redis:6379
    depends_on:
      - herald-redis

  # Redis de Herald
  herald-redis:
    image: redis:7.4-alpine
    container_name: herald-redis
    networks:
      - auth-network
    volumes:
      - herald-redis-data:/data

networks:
  auth-network:
    driver: bridge

volumes:
  warden-redis-data:
  herald-redis-data:
```

### Configuration des variables d'environnement

Créez un fichier `.env` :

```bash
# Clé d'API de Warden
WARDEN_API_KEY=your-warden-api-key-here

# Clés HMAC de Warden (format JSON)
WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'

# Secret HMAC utilisé par Stargate (correspond à la clé dans WARDEN_HMAC_KEYS)
WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef
```

### Configuration réseau

Tous les services doivent se trouver sur le même réseau Docker afin de communiquer entre eux :

- **Warden** : écoute sur le port `8081`, appelé par Stargate
- **Stargate** : écoute sur le port `8080`, sert de service forwardAuth pour Traefik
- **Herald** : écoute sur le port `8082`, appelé par Stargate

### Dépendances entre services

- **Stargate** dépend de **Warden** et **Herald**
- **Warden** dépend de **warden-redis** (facultatif, si Redis est activé)
- **Herald** dépend de **herald-redis**

### Contrôles de santé

Tous les services doivent configurer des contrôles de santé pour garantir un fonctionnement normal :

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### Recommandations pour l'environnement de production

1. **Utiliser des instances Redis indépendantes** : Warden et Herald doivent utiliser des instances Redis distinctes pour éviter les conflits de données
2. **Configurer l'authentification entre services** : l'environnement de production doit configurer mTLS ou la signature HMAC
3. **Utiliser des services de gestion de clés** : utilisez HashiCorp Vault ou un service équivalent pour gérer clés et certificats
4. **Isolation réseau** : utilisez les politiques réseau de Docker pour restreindre les accès entre services
5. **Supervision et journalisation** : mettez en place des systèmes unifiés de supervision et de collecte des journaux

### Déploiement intégré sur Kubernetes

Pour un déploiement sur Kubernetes, il est recommandé de :

1. **Utiliser des Services** : créer un Service Kubernetes pour chaque service
2. **Utiliser ConfigMap et Secret** : y stocker la configuration et les clés
3. **Utiliser NetworkPolicy** : restreindre les accès réseau entre services
4. **Utiliser Ingress** : configurer l'Ingress Traefik pour router vers Stargate

Exemple de configuration Kubernetes :

```yaml
apiVersion: v1
kind: Service
metadata:
  name: warden
spec:
  selector:
    app: warden
  ports:
    - port: 8081
      targetPort: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: ghcr.io/soulteary/warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "warden-redis:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        - name: WARDEN_HMAC_KEYS
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: hmac-keys
```

## Documentation associée

- [Documentation de configuration](CONFIGURATION.md) – Découvrez les options de configuration détaillées
- [Documentation de sécurité](SECURITY.md) – Découvrez la configuration de sécurité et les bonnes pratiques
- [Document de conception de l'architecture](ARCHITECTURE.md) – Comprendre l'architecture du système
- [Documentation de l'API](API.md) – Découvrez les interfaces de l'API et des exemples d'intégration
