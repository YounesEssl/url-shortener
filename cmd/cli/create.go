package cli

import (
	"fmt"
	"log"
	"net/url" // Pour valider le format de l'URL

	cmd2 "github.com/axellelanca/urlshortener/cmd"
	"github.com/axellelanca/urlshortener/internal/config"
	"github.com/axellelanca/urlshortener/internal/repository"
	"github.com/axellelanca/urlshortener/internal/services"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite" // Driver SQLite pour GORM
	"gorm.io/gorm"
)

// longURLFlag stockera la valeur du flag --url
var longURLFlag string

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.

Exemple:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
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

		// Appeler le LinkService et la fonction CreateLink pour créer le lien court.
		link, err := linkService.CreateLink(longURLFlag)
		if err != nil {
			log.Fatalf("FATAL: Échec de la création du lien court: %v", err)
		}

		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("URL complète: %s\n", fullShortURL)
	},
}

// init() s'exécute automatiquement lors de l'importation du package.
// Il est utilisé pour définir les flags que cette commande accepte.
func init() {
	// Définir le flag --url pour la commande create.
	CreateCmd.Flags().StringVarP(&longURLFlag, "url", "u", "", "L'URL longue à raccourcir")

	// Marquer le flag comme requis
	CreateCmd.MarkFlagRequired("url")

	// Ajouter la commande à RootCmd
	cmd2.RootCmd.AddCommand(CreateCmd)
}
