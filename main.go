package main

import (
	"github.com/axellelanca/urlshortener/cmd"
	_ "github.com/axellelanca/urlshortener/cmd/cli"    // Importe le package 'cli' pour que ses init() soient exécutés
	_ "github.com/axellelanca/urlshortener/cmd/server" // Importe le package 'server' pour que ses init() soient exécutés
)

// main est le point d'entrée principal de l'application.
// Il délègue l'exécution à la fonction Execute() de Cobra qui va gérer les sous-commandes.
func main() {
	cmd.Execute()
}
