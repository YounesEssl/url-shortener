package models

import "time"

// Link représente un lien raccourci dans la base de données.
// Les tags `gorm:"..."` définissent comment GORM doit mapper cette structure à une table SQL.
type Link struct {
	ID        uint      `gorm:"primaryKey"`                    // Clé primaire
	ShortCode string    `gorm:"uniqueIndex;size:10;not null"`  // Code court unique, indexé pour des recherches rapides
	LongURL   string    `gorm:"not null"`                      // URL longue originale
	CreatedAt time.Time `gorm:"autoCreateTime"`                // Horodatage de la création du lien
}
