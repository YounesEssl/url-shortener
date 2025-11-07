package models

import "time"

// Link représente un lien raccourci dans la base de données.
// Les tags `gorm:"..."` définissent comment GORM doit mapper cette structure à une table SQL.
type Link struct {
	ID        uint       `gorm:"primaryKey"`                        // ID est la clé primaire auto-incrémentée
	ShortCode string     `gorm:"uniqueIndex;size:10;not null"`      // ShortCode doit être unique, indexé pour des recherches rapides, taille max 10 caractères
	LongURL   string     `gorm:"not null"`                          // LongURL ne doit pas être null
	CreatedAt time.Time  `gorm:"autoCreateTime"`                    // Horodatage de la création du lien (géré automatiquement par GORM)
	IsActive  bool       `gorm:"default:true"`                      // Indicateur si le lien est actif (pour la surveillance)
	ExpiresAt *time.Time `gorm:"index"`                             // Date d'expiration optionnelle du lien (feature bonus), indexé pour des requêtes efficaces
}

// IsExpired vérifie si le lien a expiré.
// Retourne true si le lien a une date d'expiration et que cette date est dépassée.
func (l *Link) IsExpired() bool {
	// Si ExpiresAt est nil, le lien n'expire jamais
	if l.ExpiresAt == nil {
		return false
	}
	// Comparer la date d'expiration avec l'heure actuelle
	return time.Now().After(*l.ExpiresAt)
}
