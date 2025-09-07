# CheckLogs Go SDK - Version Simple

SDK Go simple pour [CheckLogs.dev](https://checklogs.dev) - Système de monitoring de logs.

## 🚀 Installation et Setup Local

### 1. Créer le projet

```bash
# Créer le répertoire
mkdir checklogs-go-sdk
cd checklogs-go-sdk

# Créer les fichiers (voir structure ci-dessous)
```

### 2. Structure des fichiers

```
checklogs-go-sdk/
├── go.mod           # Configuration du module
├── checklogs.go     # Code principal du SDK
├── README.md        # Ce fichier
└── examples/
    └── basic.go     # Exemple complet
```

### 3. Initialiser le module

```bash
# Initialiser le module Go
go mod init checklogs

# Télécharger les dépendances
go mod tidy
```

### 4. Tester le module

```bash
# Test simple (sans clé API)
go run examples/basic.go

# Test avec votre clé API
set CHECKLOGS_API_KEY=your-api-key-here
go run examples/basic.go

# Ou sur Linux/Mac
export CHECKLOGS_API_KEY=your-api-key-here
go run examples/basic.go
```

## 📖 Utilisation

### Usage basique

```go
package main

import (
    "context"
    "checklogs"
)

func main() {
    // Créer un logger
    logger := checklogs.CreateLogger("your-api-key")
    
    ctx := context.Background()
    
    // Envoyer des logs
    logger.Info(ctx, "Application démarrée")
    logger.Error(ctx, "Une erreur s'est produite", map[string]interface{}{
        "error_code": 500,
        "component": "database",
    })
}
```

### Logger avec options

```go
userID := int64(123)

options := &checklogs.Options{
    Source:        "mon-app",
    UserID:        &userID,
    Context: map[string]interface{}{
        "version": "1.0.0",
        "env": "production",
    },
    ConsoleOutput: true,
}

logger := checklogs.NewLogger("your-api-key", options)
```

### Child Logger

```go
// Logger principal
mainLogger := checklogs.CreateLogger("your-api-key")

// Child logger avec contexte spécifique
userLogger := mainLogger.Child(map[string]interface{}{
    "module": "user",
    "request_id": "req_123",
})

userLogger.Info(ctx, "Utilisateur connecté")
```

### Timer pour mesurer les performances

```go
logger := checklogs.CreateLogger("your-api-key")

// Démarrer un timer
timer := logger.Time("db-query", "Requête base de données")

// ... votre code ...

// Terminer et logger la durée
duration := timer.End()
```

## 🛠️ Commandes utiles

```bash
# Compiler le projet
go build

# Lancer les tests
go test

# Formater le code
go fmt

# Lancer l'exemple
go run examples/basic.go

# Lancer avec une clé API spécifique
go run examples/basic.go test-key your-actual-api-key
```

## 📋 Niveaux de log disponibles

- `checklogs.Debug` - Messages de debug
- `checklogs.Info` - Informations générales
- `checklogs.Warning` - Avertissements
- `checklogs.Error` - Erreurs
- `checklogs.Critical` - Erreurs critiques

## 🔧 Configuration

### Options disponibles

```go
type Options struct {
    Source        string                 // Source des logs
    UserID        *int64                 // ID utilisateur par défaut
    Context       map[string]interface{} // Contexte par défaut
    Silent        bool                   // Mode silencieux (pas d'envoi HTTP)
    ConsoleOutput bool                   // Affichage console (défaut: true)
    BaseURL       string                 // URL de l'API (défaut: CheckLogs)
    Timeout       time.Duration          // Timeout HTTP (défaut: 30s)
}
```

### Variables d'environnement

- `CHECKLOGS_API_KEY` - Votre clé API CheckLogs

## 🚨 Gestion d'erreurs

Le SDK retourne des erreurs typées :

```go
err := logger.Info(ctx, "test")
if err != nil {
    if checkLogsErr, ok := err.(*checklogs.CheckLogsError); ok {
        switch checkLogsErr.Type {
        case "ValidationError":
            // Erreur de validation
        case "NetworkError":
            // Erreur réseau
        case "APIError":
            // Erreur API (HTTP 4xx/5xx)
        }
    }
}
```

## 🔄 Queue de retry

Le SDK gère automatiquement les échecs temporaires :

```go
// Vérifier la taille de la queue
size := logger.GetRetryQueueSize()

// Forcer l'envoi des logs en attente
success := logger.FlushRetryQueue(ctx)

// Nettoyer la queue
logger.ClearRetryQueue()
```

## 📊 Exemples d'intégration

### Application web avec Gin

```go
package main

import (
    "github.com/gin-gonic/gin"
    "checklogs"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    r := gin.Default()
    
    // Middleware de logging
    r.Use(func(c *gin.Context) {
        requestLogger := logger.Child(map[string]interface{}{
            "method": c.Request.Method,
            "path": c.Request.URL.Path,
            "ip": c.ClientIP(),
        })
        
        c.Set("logger", requestLogger)
        c.Next()
    })
    
    r.GET("/users/:id", func(c *gin.Context) {
        logger := c.MustGet("logger").(*checklogs.Logger)
        userID := c.Param("id")
        
        logger.Info(c.Request.Context(), "Récupération utilisateur", map[string]interface{}{
            "user_id": userID,
        })
        
        c.JSON(200, gin.H{"user": "data"})
    })
    
    r.Run(":8080")
}
```

### Traitement de tâches en arrière-plan

```go
func processJob(jobID string) {
    logger := checklogs.CreateLogger("your-api-key")
    
    jobLogger := logger.Child(map[string]interface{}{
        "job_id": jobID,
        "worker": "background-processor",
    })
    
    timer := jobLogger.Time("job-processing", "Traitement de la tâche")
    
    ctx := context.Background()
    jobLogger.Info(ctx, "Début du traitement")
    
    // ... traitement ...
    
    duration := timer.End()
    jobLogger.Info(ctx, "Tâche terminée", map[string]interface{}{
        "status": "completed",
        "duration_seconds": duration.Seconds(),
    })
}
```

## ⚡ Tests de performance

Le SDK inclut des exemples de tests de performance :

```bash
# Test avec 1000 logs
go run examples/basic.go benchmark 1000

# Test de concurrence
go run examples/basic.go stress
```

## 🐛 Debug et développement

### Mode debug

```go
// Logger en mode silencieux pour les tests
logger := checklogs.NewLogger("", &checklogs.Options{
    Silent: true,  // Pas d'envoi HTTP
    ConsoleOutput: true,  // Affichage console uniquement
})
```

### Validation des logs

```go
// Tester la validation
err := logger.Info(ctx, strings.Repeat("A", 1025)) // Message trop long
if err != nil {
    fmt.Printf("Erreur de validation: %v\n", err)
}
```

## 📋 Checklist de déploiement

- [ ] Clé API CheckLogs configurée
- [ ] Tests unitaires passent (`go test`)
- [ ] Code formaté (`go fmt`)
- [ ] Pas d'erreurs de compilation (`go build`)
- [ ] Logs de test envoyés avec succès
- [ ] Gestion d'erreurs testée
- [ ] Queue de retry fonctionnelle

## 🔧 Troubleshooting

### Problèmes courants

**1. "API key is required"**
```bash
# Vérifier que la variable d'environnement est définie
echo $CHECKLOGS_API_KEY  # Linux/Mac
echo %CHECKLOGS_API_KEY%  # Windows
```

**2. "Network Error"**
- Vérifier la connexion internet
- Vérifier le timeout (augmenter si nécessaire)
- Vérifier l'URL de l'API

**3. "Message too long"**
- Les messages sont limités à 1024 caractères
- Tronquer ou diviser les messages longs

**4. "Source too long"**
- La source est limitée à 100 caractères
- Utiliser des noms courts et descriptifs

### Logs de debug

Pour voir les détails des requêtes HTTP, vous pouvez temporairement modifier le code pour ajouter du debug :

```go
// Dans sendLog(), avant l.httpClient.Do(req)
fmt.Printf("Envoi vers: %s\n", req.URL.String())
fmt.Printf("Headers: %v\n", req.Header)
```

## 📝 Changelog

### v1.0.0 (Version initiale)
- Logger de base avec niveaux multiples
- Support des contextes et métadonnées
- Child loggers
- Timers pour mesures de performance
- Queue de retry automatique
- Gestion d'erreurs typées
- Mode silencieux pour développement
- Validation des données

## 🤝 Contribution

Pour contribuer au développement :

1. Forker le repository
2. Créer une branche feature
3. Tester les modifications
4. Créer une pull request

## 📄 License

MIT License - Voir le fichier LICENSE pour les détails.

## 📞 Support

- Documentation: [https://docs.checklogs.dev](https://docs.checklogs.dev)
- Issues: GitHub Issues
- Email: [support@checklogs.dev](mailto:support@checklogs.dev)