package errors

import "fmt"

// ErrLinkNotFound est retournée quand un lien n'existe pas dans la base de données.
type ErrLinkNotFound struct {
	ShortCode string
}

func (e *ErrLinkNotFound) Error() string {
	return fmt.Sprintf("lien avec le code '%s' non trouvé", e.ShortCode)
}

// ErrCodeGenerationFailed est retournée quand la génération d'un code unique échoue.
type ErrCodeGenerationFailed struct {
	Attempts int
}

func (e *ErrCodeGenerationFailed) Error() string {
	return fmt.Sprintf("impossible de générer un code unique après %d tentatives", e.Attempts)
}

// ErrInvalidURL est retournée quand une URL fournie est invalide.
type ErrInvalidURL struct {
	URL string
}

func (e *ErrInvalidURL) Error() string {
	return fmt.Sprintf("URL invalide: %s", e.URL)
}
