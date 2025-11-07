package middleware

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IPRateLimiter gère le rate limiting par adresse IP.
// Cette structure fait partie des features bonus et permet de limiter le nombre de requêtes
// qu'une même IP peut effectuer dans un intervalle de temps donné.
type IPRateLimiter struct {
	ips        map[string]*IPLimitInfo // Map des IPs avec leurs informations de limitation
	mu         sync.RWMutex            // Mutex pour protéger l'accès concurrent à la map
	maxRequest int                     // Nombre maximum de requêtes autorisées
	window     time.Duration           // Fenêtre de temps pour le rate limiting
}

// IPLimitInfo contient les informations de limitation pour une IP spécifique.
type IPLimitInfo struct {
	count      int       // Nombre de requêtes effectuées dans la fenêtre actuelle
	resetTime  time.Time // Moment où le compteur sera réinitialisé
	lastAccess time.Time // Dernière fois que cette IP a fait une requête
}

// NewIPRateLimiter crée une nouvelle instance de rate limiter.
// maxRequest: nombre maximum de requêtes autorisées par IP
// windowMinutes: durée de la fenêtre de temps en minutes
func NewIPRateLimiter(maxRequest int, windowMinutes int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:        make(map[string]*IPLimitInfo),
		maxRequest: maxRequest,
		window:     time.Duration(windowMinutes) * time.Minute,
	}

	// Lancer une goroutine pour nettoyer périodiquement les anciennes entrées
	// Cela évite que la map grandisse indéfiniment
	go limiter.cleanupOldEntries()

	return limiter
}

// cleanupOldEntries nettoie périodiquement les entrées IP qui n'ont pas été utilisées depuis longtemps.
// Cette méthode s'exécute dans une goroutine séparée.
func (rl *IPRateLimiter) cleanupOldEntries() {
	ticker := time.NewTicker(10 * time.Minute) // Nettoyage toutes les 10 minutes
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		// Supprimer les entrées qui n'ont pas été accédées depuis plus de 2 fois la fenêtre de temps
		for ip, info := range rl.ips {
			if now.Sub(info.lastAccess) > rl.window*2 {
				delete(rl.ips, ip)
			}
		}
		rl.mu.Unlock()
		log.Printf("[RATE LIMITER] Nettoyage effectué. Nombre d'IPs suivies: %d", len(rl.ips))
	}
}

// isAllowed vérifie si une IP est autorisée à faire une requête.
// Retourne true si la requête est autorisée, false sinon.
// Met également à jour le compteur de requêtes pour cette IP.
func (rl *IPRateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Récupérer ou créer les informations pour cette IP
	info, exists := rl.ips[ip]
	if !exists {
		// Première requête de cette IP
		rl.ips[ip] = &IPLimitInfo{
			count:      1,
			resetTime:  now.Add(rl.window),
			lastAccess: now,
		}
		return true
	}

	// Mettre à jour le dernier accès
	info.lastAccess = now

	// Vérifier si la fenêtre de temps est expirée
	if now.After(info.resetTime) {
		// Réinitialiser le compteur
		info.count = 1
		info.resetTime = now.Add(rl.window)
		return true
	}

	// Vérifier si le nombre maximum de requêtes est atteint
	if info.count >= rl.maxRequest {
		log.Printf("[RATE LIMITER] IP %s a dépassé la limite (%d requêtes en %v)", ip, rl.maxRequest, rl.window)
		return false
	}

	// Incrémenter le compteur
	info.count++
	return true
}

// getRemainingRequests retourne le nombre de requêtes restantes pour une IP.
func (rl *IPRateLimiter) getRemainingRequests(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	info, exists := rl.ips[ip]
	if !exists {
		return rl.maxRequest
	}

	now := time.Now()
	if now.After(info.resetTime) {
		return rl.maxRequest
	}

	remaining := rl.maxRequest - info.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// getResetTime retourne le moment où le compteur sera réinitialisé pour une IP.
func (rl *IPRateLimiter) getResetTime(ip string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	info, exists := rl.ips[ip]
	if !exists {
		return time.Now().Add(rl.window)
	}

	now := time.Now()
	if now.After(info.resetTime) {
		return now.Add(rl.window)
	}

	return info.resetTime
}

// RateLimitMiddleware crée un middleware Gin pour le rate limiting par IP.
// Ce middleware doit être appliqué aux routes que vous souhaitez protéger.
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupérer l'adresse IP du client
		ip := c.ClientIP()

		// Vérifier si l'IP est autorisée
		if !limiter.isAllowed(ip) {
			// L'IP a dépassé la limite
			resetTime := limiter.getResetTime(ip)
			secondsUntilReset := int(time.Until(resetTime).Seconds())

			// Ajouter des headers informatifs
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.maxRequest))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
			c.Header("Retry-After", fmt.Sprintf("%d", secondsUntilReset))

			// Retourner une erreur 429 Too Many Requests
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":             "Trop de requêtes. Veuillez réessayer plus tard.",
				"retry_after":       secondsUntilReset,
				"reset_at":          resetTime.Format(time.RFC3339),
				"max_requests":      limiter.maxRequest,
				"window_minutes":    int(limiter.window.Minutes()),
			})
			c.Abort() // Arrêter le traitement de la requête
			return
		}

		// L'IP est autorisée, ajouter des headers informatifs
		remaining := limiter.getRemainingRequests(ip)
		resetTime := limiter.getResetTime(ip)
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.maxRequest))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		// Continuer le traitement de la requête
		c.Next()
	}
}
