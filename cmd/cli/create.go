package cli

import (
	"fmt"
	"log"
	"net/url" // Pour valider le format de l'URL

	cmd2 "github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/models"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite" // Driver SQLite pour GORM
	"gorm.io/gorm"
)

// longURLFlag stockera la valeur du flag --url
var longURLFlag string

// customAliasFlag stockera la valeur du flag --alias (optionnel, feature bonus)
var customAliasFlag string

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.
Vous pouvez optionnellement spécifier un alias personnalisé avec --alias (feature bonus).

Exemples:
  url-shortener create --url="https://www.google.com/search?q=go+lang"
  url-shortener create --url="https://www.google.com" --alias="mon-lien-google"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Valider que le flag --url a été fourni.
		if longURLFlag == "" {
			log.Fatalf("FATAL: Le flag --url est requis")
		}

		// Validation basique du format de l'URL avec le package url et la fonction ParseRequestURI
		if _, err := url.ParseRequestURI(longURLFlag); err != nil {
			log.Fatalf("FATAL: URL invalide: %v", err)
		}

		// Charger la configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("FATAL: Impossible de charger la configuration: %v", err)
		}

		// Initialiser la connexion à la base de données SQLite.
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Impossible de se connecter à la base de données: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}

		// S'assurer que la connexion est fermée à la fin de l'exécution de la commande
		defer func() {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Attention: Erreur lors de la fermeture de la connexion: %v", err)
			}
		}()

		// Initialiser les repositories et services nécessaires NewLinkRepository & NewLinkService
		linkRepo := repository.NewLinkRepository(db)
		linkService := services.NewLinkService(linkRepo)

		// Vérifier si un alias personnalisé a été fourni (feature bonus)
		var link *models.Link
		if customAliasFlag != "" {
			// Créer le lien avec l'alias personnalisé
			fmt.Printf("Création d'un lien avec l'alias personnalisé: %s\n", customAliasFlag)
			link, err = linkService.CreateLinkWithCustomAlias(longURLFlag, customAliasFlag)
			if err != nil {
				log.Fatalf("FATAL: Échec de la création du lien avec alias personnalisé: %v", err)
			}
		} else {
			// Créer le lien avec un code généré automatiquement
			link, err = linkService.CreateLink(longURLFlag)
			if err != nil {
				log.Fatalf("FATAL: Échec de la création du lien court: %v", err)
			}
		}

		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("URL complète: %s\n", fullShortURL)
		if link.IsCustom {
			fmt.Printf("Type: Alias personnalisé ✨\n")
		}
	},
}

// init() s'exécute automatiquement lors de l'importation du package.
// Il est utilisé pour définir les flags que cette commande accepte.
func init() {
	// Définir le flag --url pour la commande create.
	CreateCmd.Flags().StringVarP(&longURLFlag, "url", "u", "", "L'URL longue à raccourcir")

	// Définir le flag --alias pour spécifier un alias personnalisé (optionnel, feature bonus)
	CreateCmd.Flags().StringVarP(&customAliasFlag, "alias", "a", "", "Alias personnalisé pour l'URL courte (optionnel)")

	// Marquer le flag --url comme requis
	CreateCmd.MarkFlagRequired("url")

	// Ajouter la commande à RootCmd
	cmd2.RootCmd.AddCommand(CreateCmd)
}
