package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"checklogs"
)

func main() {
	fmt.Println("🚀 CheckLogs Go SDK - Exemple Simple")
	fmt.Println("===================================")

	// Récupérer la clé API depuis les variables d'environnement
	apiKey := os.Getenv("CHECKLOGS_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  Variable CHECKLOGS_API_KEY non définie")
		fmt.Println("   Mode démo activé (pas d'envoi réel)")
		apiKey = "demo-key"
	}

	// 1. Exemple de base
	exempleBase(apiKey)

	// 2. Logger avec options
	exempleAvecOptions(apiKey)

	// 3. Child logger
	exempleChildLogger(apiKey)

	// 4. Timer
	exempleTimer(apiKey)

	// 5. Gestion d'erreurs
	exempleGestionErreurs(apiKey)

	// 6. Queue de retry
	exempleRetryQueue(apiKey)

	fmt.Println("\n✅ Tous les exemples terminés avec succès!")
}

// Exemple de base
func exempleBase(apiKey string) {
	fmt.Println("\n📝 1. Exemple de base")
	fmt.Println("--------------------")

	// Créer un logger simple
	logger := checklogs.CreateLogger(apiKey)

	ctx := context.Background()

	// Envoyer différents types de logs
	logger.Debug(ctx, "Message de debug pour le développement")
	logger.Info(ctx, "Application démarrée avec succès")
	logger.Warning(ctx, "Attention: Espace disque faible")
	logger.Error(ctx, "Erreur de connexion à la base de données", map[string]interface{}{
		"error_code": "DB_CONNECTION_FAILED",
		"retry_count": 3,
	})
	logger.Critical(ctx, "Échec critique du système", map[string]interface{}{
		"component": "auth_service",
		"severity": "high",
	})

	fmt.Println("✅ Logs de base envoyés")
}

// Logger avec options
func exempleAvecOptions(apiKey string) {
	fmt.Println("\n⚙️  2. Logger avec options")
	fmt.Println("------------------------")

	userID := int64(12345)

	// Créer un logger avec des options personnalisées
	options := &checklogs.Options{
		Source:        "exemple-app",
		UserID:        &userID,
		Context: map[string]interface{}{
			"environment": "development",
			"version":     "1.0.0",
			"service":     "api-server",
		},
		ConsoleOutput: true,
		Silent:        false,
	}

	logger := checklogs.NewLogger(apiKey, options)
	ctx := context.Background()

	logger.Info(ctx, "Logger configuré avec options personnalisées")
	logger.Error(ctx, "Erreur avec contexte enrichi", map[string]interface{}{
		"request_id": "req_123456789",
		"user_action": "login",
		"ip_address": "192.168.1.100",
	})

	fmt.Println("✅ Logger avec options configuré")
}

// Child logger
func exempleChildLogger(apiKey string) {
	fmt.Println("\n👶 3. Child Logger")
	fmt.Println("------------------")

	// Logger principal
	mainLogger := checklogs.NewLogger(apiKey, &checklogs.Options{
		Source: "microservice",
		Context: map[string]interface{}{
			"service": "user-management",
			"version": "2.0.0",
		},
	})

	ctx := context.Background()

	// Créer des child loggers pour différents modules
	authLogger := mainLogger.Child(map[string]interface{}{
		"module": "authentication",
	})

	userLogger := mainLogger.Child(map[string]interface{}{
		"module": "user-crud",
	})

	// Utiliser les child loggers
	authLogger.Info(ctx, "Tentative d'authentification", map[string]interface{}{
		"username": "john.doe",
		"method":   "oauth2",
	})

	userLogger.Info(ctx, "Profil utilisateur mis à jour", map[string]interface{}{
		"user_id": 12345,
		"fields":  []string{"email", "phone"},
	})

	// Child logger imbriqué
	requestLogger := authLogger.Child(map[string]interface{}{
		"request_id": generateRequestID(),
		"session_id": generateSessionID(),
	})

	requestLogger.Warning(ctx, "Multiples tentatives de connexion échouées détectées")

	fmt.Println("✅ Child loggers utilisés avec succès")
}

// Timer
func exempleTimer(apiKey string) {
	fmt.Println("\n⏱️  4. Timer")
	fmt.Println("------------")

	logger := checklogs.NewLogger(apiKey, &checklogs.Options{
		Source: "timer-example",
	})

	// Simuler une opération de base de données
	timer := logger.Time("db-query", "Exécution d'une requête de base de données")

	// Simuler du travail
	simulerTravailDB()

	duration := timer.End()
	fmt.Printf("⏱️  Opération terminée en %v\n", duration)

	// Timers multiples
	timer1 := logger.Time("api-call", "Appel API externe")
	timer2 := logger.Time("data-processing", "Traitement des données")

	simulerAppelAPI()
	duration1 := timer1.End()

	simulerTraitementDonnees()
	duration2 := timer2.End()

	fmt.Printf("⏱️  Appel API: %v, Traitement: %v\n", duration1, duration2)
}

// Gestion d'erreurs
func exempleGestionErreurs(apiKey string) {
	fmt.Println("\n🚨 5. Gestion d'erreurs")
	fmt.Println("----------------------")

	logger := checklogs.NewLogger(apiKey, nil)
	ctx := context.Background()

	// Test avec un message trop long
	messageTropLong := make([]byte, 1025) // Dépasse la limite de 1024
	for i := range messageTropLong {
		messageTropLong[i] = 'A'
	}

	err := logger.Info(ctx, string(messageTropLong))
	if err != nil {
		if checkLogsErr, ok := err.(*checklogs.CheckLogsError); ok {
			fmt.Printf("🔴 Erreur de validation: %s\n", checkLogsErr.Message)
		}
	}

	// Test avec source trop longue
	longueSource := make([]byte, 101) // Dépasse la limite de 100
	for i := range longueSource {
		longueSource[i] = 'B'
	}

	loggerSourceLongue := checklogs.NewLogger(apiKey, &checklogs.Options{
		Source: string(longueSource),
	})

	err = loggerSourceLongue.Info(ctx, "Test avec source trop longue")
	if err != nil {
		if checkLogsErr, ok := err.(*checklogs.CheckLogsError); ok {
			fmt.Printf("🔴 Erreur de validation: %s\n", checkLogsErr.Message)
		}
	}

	fmt.Println("✅ Gestion d'erreurs testée")
}

// Queue de retry
func exempleRetryQueue(apiKey string) {
	fmt.Println("\n🔄 6. Queue de retry")
	fmt.Println("-------------------")

	// Créer un logger avec timeout très court pour forcer les erreurs
	logger := checklogs.NewLogger(apiKey, &checklogs.Options{
		Source:  "retry-test",
		Timeout: 1 * time.Millisecond, // Timeout très court
	})

	ctx := context.Background()

	// Envoyer des logs qui vont probablement échouer
	for i := 0; i < 3; i++ {
		err := logger.Info(ctx, fmt.Sprintf("Log de test %d", i+1), map[string]interface{}{
			"attempt": i + 1,
			"test":    true,
		})

		if err != nil {
			fmt.Printf("⚠️  Log %d a échoué (attendu): %v\n", i+1, err)
		}
	}

	// Vérifier la taille de la queue
	queueSize := logger.GetRetryQueueSize()
	fmt.Printf("📊 Taille de la queue de retry: %d\n", queueSize)

	// Créer un nouveau logger avec timeout normal pour flush
	normalLogger := checklogs.NewLogger(apiKey, &checklogs.Options{
		Source: "retry-flush",
	})

	// Simuler un flush (en réalité, on ne peut pas transférer la queue)
	flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	success := normalLogger.FlushRetryQueue(flushCtx)
	fmt.Printf("✅ Flush terminé, %d logs envoyés avec succès\n", success)

	// Nettoyer la queue
	logger.ClearRetryQueue()
	fmt.Printf("🧹 Queue nettoyée, nouvelle taille: %d\n", logger.GetRetryQueueSize())
}

// Fonctions utilitaires

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

func simulerTravailDB() {
	time.Sleep(100 * time.Millisecond)
}

func simulerAppelAPI() {
	time.Sleep(200 * time.Millisecond)
}

func simulerTraitementDonnees() {
	time.Sleep(150 * time.Millisecond)
}

// Fonction principale alternative avec arguments
func init() {
	// Test avec clé API fournie en argument
	if len(os.Args) > 1 && os.Args[1] == "test-key" && len(os.Args) > 2 {
		testKey := os.Args[2]
		os.Setenv("CHECKLOGS_API_KEY", testKey)
		fmt.Printf("🔑 Test avec clé API: %s...\n", testKey[:min(len(testKey), 10)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}