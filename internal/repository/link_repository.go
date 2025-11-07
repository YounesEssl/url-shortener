package repository

import (
    "github.com/axellelanca/urlshortener/internal/models"
    "gorm.io/gorm"
)

// LinkRepository définit les opérations d'accès aux données pour les liens
type LinkRepository interface {
    CreateLink(link *models.Link) error
    GetLinkByShortCode(shortCode string) (*models.Link, error)
    GetAllLinks() ([]models.Link, error)
    CountClicksByLinkID(linkID uint) (int, error)
}

// GormLinkRepository est l'implémentation de LinkRepository utilisant GORM.
type GormLinkRepository struct {
    db *gorm.DB
}

// NewLinkRepository crée et retourne une nouvelle instance de GormLinkRepository.
// Cette fonction retourne *GormLinkRepository, qui implémente l'interface LinkRepository.
func NewLinkRepository(db *gorm.DB) *GormLinkRepository {
    return &GormLinkRepository{db: db}
}

// CreateLink insère un nouveau lien dans la base de données.
func (r *GormLinkRepository) CreateLink(link *models.Link) error {
    return r.db.Create(link).Error
}

// GetLinkByShortCode récupère un lien de la base de données en utilisant son shortCode.
// Il renvoie gorm.ErrRecordNotFound si aucun lien n'est trouvé avec ce shortCode.
func (r *GormLinkRepository) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	var link models.Link
    if err := r.db.Where("short_code = ?", shortCode).First(&link).Error; err != nil {
        return nil, err
    }
    return &link, nil
}

// GetAllLinks récupère tous les liens de la base de données.
// Cette méthode est utilisée par le moniteur d'URLs.
func (r *GormLinkRepository) GetAllLinks() ([]models.Link, error) {
	var links []models.Link
    if err := r.db.Find(&links).Error; err != nil {
        return nil, err
    }
    return links, nil
}

// CountClicksByLinkID compte le nombre total de clics pour un ID de lien donné.
func (r *GormLinkRepository) CountClicksByLinkID(linkID uint) (int, error) {
	var count int64 // GORM retourne un int64 pour les comptes
    if err := r.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&count).Error; err != nil {
        return 0, err
    }
	return int(count), nil
}
