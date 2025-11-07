package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Pour gérer gorm.ErrRecordNotFound
)

// ClickEventsChannel est le channel global (ou injecté) utilisé pour envoyer les événements de clic
// aux workers asynchrones. Il est bufferisé pour ne pas bloquer les requêtes de redirection.
var ClickEventsChannel chan models.ClickEvent

// SetupRoutes configure toutes les routes de l'API Gin et injecte les dépendances nécessaires
func SetupRoutes(router *gin.Engine, linkService *services.LinkService, cfg *config.Config) {
	// Le channel est initialisé ici.
	if ClickEventsChannel == nil {
		// Créer le channel bufferisé
		// La taille du buffer doit être configurable via la donnée récupérée avec Viper
		ClickEventsChannel = make(chan models.ClickEvent, 1000)
	}

	// Route de Health Check , /health
	router.GET("/health", HealthCheckHandler)

	// Routes de l'API
	// Doivent être au format /api/v1/
	// POST /links
	// GET /links/:shortCode/stats
	api := router.Group("/api/v1")
	{
		api.POST("/links", CreateShortLinkHandler(linkService, cfg))
		api.GET("/links/:shortCode/stats", GetLinkStatsHandler(linkService))
	}

	// Route de Redirection (au niveau racine pour les short codes)
	router.GET("/:shortCode", RedirectHandler(linkService))
}

// HealthCheckHandler gère la route /health pour vérifier l'état du service.
func HealthCheckHandler(c *gin.Context) {
	// Retourner simplement du JSON avec un StatusOK, {"status": "ok"}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateLinkRequest représente le corps de la requête JSON pour la création d'un lien.
type CreateLinkRequest struct {
	LongURL           string `json:"long_url" binding:"required,url"` // 'binding:required' pour validation, 'url' pour format URL
	ExpirationMinutes int    `json:"expiration_minutes,omitempty"`    // Durée de vie du lien en minutes (optionnel, feature bonus)
}

// CreateShortLinkHandler gère la création d'une URL courte.
func CreateShortLinkHandler(linkService *services.LinkService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateLinkRequest
		// Tente de lier le JSON de la requête à la structure CreateLinkRequest.
		// Gin gère la validation 'binding'.
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}

		var link *models.Link
		var err error

		// Vérifier si une durée d'expiration a été fournie (feature bonus)
		if req.ExpirationMinutes > 0 {
			// Créer le lien avec expiration
			log.Printf("Création d'un lien avec expiration: %d minutes", req.ExpirationMinutes)
			link, err = linkService.CreateLinkWithExpiration(req.LongURL, req.ExpirationMinutes)
		} else {
			// Créer le lien sans expiration
			link, err = linkService.CreateLink(req.LongURL)
		}

		if err != nil {
			log.Printf("Error creating link: %v", err)
			// Si l'erreur concerne une durée d'expiration invalide, retourner un BadRequest
			if req.ExpirationMinutes > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
			}
			return
		}

		// Préparer la réponse JSON
		response := gin.H{
			"short_code":     link.ShortCode,
			"long_url":       link.LongURL,
			"full_short_url": cfg.Server.BaseURL + "/" + link.ShortCode,
		}

		// Ajouter la date d'expiration si le lien expire
		if link.ExpiresAt != nil {
			response["expires_at"] = link.ExpiresAt.Format(time.RFC3339)
			response["expires_in_minutes"] = int(time.Until(*link.ExpiresAt).Minutes())
		}

		c.JSON(http.StatusCreated, response)
	}
}

// RedirectHandler gère la redirection d'une URL courte vers l'URL longue et l'enregistrement asynchrone des clics.
// Vérifie également si le lien a expiré (feature bonus).
func RedirectHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère le shortCode de l'URL avec c.Param
		shortCode := c.Param("shortCode")

		// Récupérer l'URL longue associée au shortCode depuis le linkService (GetLinkByShortCode)
		link, err := linkService.GetLinkByShortCode(shortCode)

		if err != nil {
			// Si le lien n'est pas trouvé, retourner HTTP 404 Not Found.
			// Utiliser errors.Is et l'erreur Gorm
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short code not found"})
				return
			}
			// Gérer d'autres erreurs potentielles de la base de données ou du service
			log.Printf("Error retrieving link for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Vérifier si le lien a expiré (feature bonus)
		if link.IsExpired() {
			log.Printf("Link %s has expired (expired at: %v)", shortCode, link.ExpiresAt)
			c.JSON(http.StatusGone, gin.H{
				"error":      "This link has expired",
				"expired_at": link.ExpiresAt.Format(time.RFC3339),
			})
			return
		}

		// Créer un ClickEvent avec les informations pertinentes.
		clickEvent := models.ClickEvent{
			LinkID:    link.ID,
			Timestamp: time.Now(),
			UserAgent: c.Request.UserAgent(),
			IPAddress: c.ClientIP(),
		}

		// Envoyer le ClickEvent dans le ClickEventsChannel avec le Multiplexage.
		// Utilise un `select` avec un `default` pour éviter de bloquer si le channel est plein.
		select {
		case ClickEventsChannel <- clickEvent:
			// Événement envoyé avec succès
		default:
			log.Printf("Warning: ClickEventsChannel is full, dropping click event for %s.", shortCode)
		}

		// Effectuer la redirection HTTP 302 (StatusFound) vers l'URL longue.
		c.Redirect(http.StatusFound, link.LongURL)
	}
}

// GetLinkStatsHandler gère la récupération des statistiques pour un lien spécifique.
func GetLinkStatsHandler(linkService *services.LinkService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupère le shortCode de l'URL avec c.Param
		shortCode := c.Param("shortCode")

		// Appeler le LinkService pour obtenir le lien et le nombre total de clics.
		link, totalClicks, err := linkService.GetLinkStats(shortCode)
		if err != nil {
			// Gérer le cas où le lien n'est pas trouvé.
			// toujours avec l'erreur Gorm ErrRecordNotFound
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Short code not found"})
				return
			}
			// Gérer d'autres erreurs
			log.Printf("Error retrieving stats for %s: %v", shortCode, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		// Retourne les statistiques dans la réponse JSON.
		c.JSON(http.StatusOK, gin.H{
			"short_code":   link.ShortCode,
			"long_url":     link.LongURL,
			"total_clicks": totalClicks,
		})
	}
}
