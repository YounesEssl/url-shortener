package config

import (
	"fmt"
	"log" // Pour logger les informations ou erreurs de chargement de config

	"github.com/spf13/viper" // La bibliothèque pour la gestion de configuration
)

// Config est la structure principale qui mappe l'intégralité de la configuration de l'application.
// Les tags `mapstructure` sont utilisés par Viper pour mapper les clés du fichier de config
// (ou des variables d'environnement) aux champs de la structure Go.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Analytics   AnalyticsConfig   `mapstructure:"analytics"`
	Monitor     MonitorConfig     `mapstructure:"monitor"`
	RateLimiter RateLimiterConfig `mapstructure:"rate_limiter"` // Configuration du rate limiting (feature bonus)
}

// ServerConfig contient la configuration du serveur web Gin.
type ServerConfig struct {
	Port    int    `mapstructure:"port"`
	BaseURL string `mapstructure:"base_url"`
}

// DatabaseConfig contient la configuration de la base de données.
type DatabaseConfig struct {
	Name string `mapstructure:"name"`
}

// AnalyticsConfig contient la configuration des analytics asynchrones.
type AnalyticsConfig struct {
	BufferSize  int `mapstructure:"buffer_size"`
	WorkerCount int `mapstructure:"worker_count"`
}

// MonitorConfig contient la configuration du moniteur d'URLs.
type MonitorConfig struct {
	IntervalMinutes int `mapstructure:"interval_minutes"`
}

// RateLimiterConfig contient la configuration du rate limiting (feature bonus).
type RateLimiterConfig struct {
	Enabled       bool `mapstructure:"enabled"`        // Activer ou désactiver le rate limiting
	MaxRequests   int  `mapstructure:"max_requests"`   // Nombre maximum de requêtes par IP
	WindowMinutes int  `mapstructure:"window_minutes"` // Fenêtre de temps en minutes
}

// LoadConfig charge la configuration de l'application en utilisant Viper.
// Elle recherche un fichier 'config.yaml' dans le dossier 'configs/'.
// Elle définit également des valeurs par défaut si le fichier de config est absent ou incomplet.
func LoadConfig() (*Config, error) {
	// Spécifie le chemin où Viper doit chercher les fichiers de config.
	// on cherche dans le dossier 'configs' relatif au répertoire d'exécution.
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Spécifie le nom du fichier de config (sans l'extension).
	viper.SetConfigName("config")

	// Spécifie le type de fichier de config.
	viper.SetConfigType("yaml")

	// Définir les valeurs par défaut pour toutes les options de configuration.
	// Ces valeurs seront utilisées si les clés correspondantes ne sont pas trouvées dans le fichier de config
	// ou si le fichier n'existe pas.
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.base_url", "http://localhost:8080")
	viper.SetDefault("database.name", "url_shortener.db")
	viper.SetDefault("analytics.buffer_size", 1000)
	viper.SetDefault("analytics.worker_count", 5)
	viper.SetDefault("monitor.interval_minutes", 5)
	// Valeurs par défaut pour le rate limiting (feature bonus)
	viper.SetDefault("rate_limiter.enabled", true)
	viper.SetDefault("rate_limiter.max_requests", 10)
	viper.SetDefault("rate_limiter.window_minutes", 1)

	// Lire le fichier de configuration.
	if err := viper.ReadInConfig(); err != nil {
		// Si le fichier n'est pas trouvé, on continue avec les valeurs par défaut
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Fichier de configuration non trouvé. Utilisation des valeurs par défaut.")
		} else {
			// Autre erreur de lecture
			return nil, fmt.Errorf("erreur lors de la lecture du fichier de configuration: %w", err)
		}
	} else {
		log.Printf("Fichier de configuration chargé: %s", viper.ConfigFileUsed())
	}

	// Démapper (unmarshal) la configuration lue (ou les valeurs par défaut) dans la structure Config.
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("erreur lors du démappage de la configuration: %w", err)
	}

	// Log  pour vérifier la config chargée
	log.Printf("Configuration loaded: Server Port=%d, DB Name=%s, Analytics Buffer=%d, Monitor Interval=%dmin",
		cfg.Server.Port, cfg.Database.Name, cfg.Analytics.BufferSize, cfg.Monitor.IntervalMinutes)

	return &cfg, nil // Retourne la configuration chargée
}
