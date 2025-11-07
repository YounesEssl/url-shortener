package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"regexp"

	"gorm.io/gorm" // Nécessaire pour la gestion spécifique de gorm.ErrRecordNotFound

	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository" // Importe le package repository
)

// Définition du jeu de caractères pour la génération des codes courts.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// LinkService est une structure qui fournit des méthodes pour la logique métier des liens.
// Elle détient linkRepo qui est une référence vers une interface LinkRepository.
// IMPORTANT : Le champ doit être du type de l'interface (non-pointeur).
type LinkService struct {
	linkRepo repository.LinkRepository
}

// NewLinkService crée et retourne une nouvelle instance de LinkService.
func NewLinkService(linkRepo repository.LinkRepository) *LinkService {
	return &LinkService{
		linkRepo: linkRepo,
	}
}

// GenerateShortCode génère un code court aléatoire d'une longueur spécifiée.
// Il utilise le package 'crypto/rand' pour éviter la prévisibilité.
func (s *LinkService) GenerateShortCode(length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("error generating random number: %w", err)
		}
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result), nil
}

// CreateLink crée un nouveau lien raccourci.
// Il génère un code court unique, puis persiste le lien dans la base de données.
func (s *LinkService) CreateLink(longURL string) (*models.Link, error) {
	// Implémenter la logique de retry pour générer un code court unique.
	// Essayez de générer un code, vérifiez s'il existe déjà en base, et retentez si une collision est trouvée.
	// Limitez le nombre de tentatives pour éviter une boucle infinie.

	var shortCode string
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		// Génère un code de 6 caractères
		code, err := s.GenerateShortCode(6)
		if err != nil {
			return nil, fmt.Errorf("error generating short code: %w", err)
		}

		// Vérifie si le code généré existe déjà en base de données
		_, err = s.linkRepo.GetLinkByShortCode(code)

		if err != nil {
			// Si l'erreur est 'record not found' de GORM, cela signifie que le code est unique.
			if errors.Is(err, gorm.ErrRecordNotFound) {
				shortCode = code // Le code est unique, on peut l'utiliser
				break            // Sort de la boucle de retry
			}
			// Si c'est une autre erreur de base de données, retourne l'erreur.
			return nil, fmt.Errorf("database error checking short code uniqueness: %w", err)
		}

		// Si aucune erreur (le code a été trouvé), cela signifie une collision.
		log.Printf("Short code '%s' already exists, retrying generation (%d/%d)...", code, i+1, maxRetries)
		// La boucle continuera pour générer un nouveau code.
	}

	// Si après toutes les tentatives, aucun code unique n'a été trouvé
	if shortCode == "" {
		return nil, errors.New("failed to generate unique short code after multiple retries")
	}

	// Crée une nouvelle instance du modèle Link.
	link := &models.Link{
		ShortCode: shortCode,
		LongURL:   longURL,
	}

	// Persiste le nouveau lien dans la base de données via le repository
	err := s.linkRepo.CreateLink(link)
	if err != nil {
		return nil, fmt.Errorf("error creating link in database: %w", err)
	}

	// Retourne le lien créé
	return link, nil
}

// GetLinkByShortCode récupère un lien via son code court.
// Il délègue l'opération de recherche au repository.
func (s *LinkService) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	// Récupérer un lien par son code court en utilisant s.linkRepo.GetLinkByShortCode.
	// Retourner le lien trouvé ou une erreur si non trouvé/problème DB.
	return s.linkRepo.GetLinkByShortCode(shortCode)
}

// GetLinkStats récupère les statistiques pour un lien donné (nombre total de clics).
// Il interagit avec le LinkRepository pour obtenir le lien, puis avec le ClickRepository
func (s *LinkService) GetLinkStats(shortCode string) (*models.Link, int, error) {
	// Récupérer le lien par son shortCode
	link, err := s.linkRepo.GetLinkByShortCode(shortCode)
	if err != nil {
		return nil, 0, err
	}

	// Compter le nombre de clics pour ce LinkID
	count, err := s.linkRepo.CountClicksByLinkID(link.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting clicks: %w", err)
	}

	// on retourne les 3 valeurs
	return link, count, nil
}

// CreateLinkWithCustomAlias crée un nouveau lien raccourci avec un alias personnalisé fourni par l'utilisateur.
// Cette méthode fait partie des features bonus et permet aux utilisateurs de choisir leur propre code court.
// Elle valide que l'alias respecte certaines règles (longueur, caractères autorisés) et qu'il n'existe pas déjà.
func (s *LinkService) CreateLinkWithCustomAlias(longURL, customAlias string) (*models.Link, error) {
	// Validation de l'alias personnalisé
	// 1. Vérifier que l'alias n'est pas vide
	if customAlias == "" {
		return nil, errors.New("l'alias personnalisé ne peut pas être vide")
	}

	// 2. Vérifier la longueur de l'alias (entre 3 et 20 caractères)
	if len(customAlias) < 3 || len(customAlias) > 20 {
		return nil, errors.New("l'alias personnalisé doit contenir entre 3 et 20 caractères")
	}

	// 3. Vérifier que l'alias ne contient que des caractères alphanumériques et des tirets
	// On utilise une regex pour valider le format
	validAliasPattern := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	if !validAliasPattern.MatchString(customAlias) {
		return nil, errors.New("l'alias personnalisé ne peut contenir que des lettres, chiffres et tirets")
	}

	// 4. Vérifier que l'alias n'est pas un mot réservé (pour éviter les conflits avec les routes API)
	reservedWords := []string{"api", "health", "stats", "admin", "create", "delete"}
	for _, reserved := range reservedWords {
		if customAlias == reserved {
			return nil, fmt.Errorf("l'alias '%s' est un mot réservé et ne peut pas être utilisé", customAlias)
		}
	}

	// 5. Vérifier que l'alias n'existe pas déjà en base de données
	existingLink, err := s.linkRepo.GetLinkByShortCode(customAlias)
	if err == nil && existingLink != nil {
		// Si aucune erreur et qu'un lien existe, cela signifie que l'alias est déjà pris
		return nil, fmt.Errorf("l'alias '%s' est déjà utilisé, veuillez en choisir un autre", customAlias)
	}

	// Si l'erreur n'est pas 'record not found', c'est une erreur de base de données
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("erreur lors de la vérification de l'alias: %w", err)
	}

	// L'alias est valide et disponible, on peut créer le lien
	link := &models.Link{
		ShortCode: customAlias,
		LongURL:   longURL,
		IsCustom:  true, // Marquer ce lien comme ayant un alias personnalisé
	}

	// Persister le lien dans la base de données
	err = s.linkRepo.CreateLink(link)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du lien avec alias personnalisé: %w", err)
	}

	log.Printf("Lien créé avec succès avec l'alias personnalisé '%s'", customAlias)
	return link, nil
}
