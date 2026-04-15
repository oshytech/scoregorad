package domain

const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// Page agrupa los parámetros de paginación offset-based.
// Implementa paginación por offset, no por cursor —
// el nombre anterior (CursorPage) era incorrecto.
type Page struct {
	Limit  int
	Offset int
}

// NewPage crea una página validando los límites.
func NewPage(limit, offset int) Page {
	if limit <= 0 || limit > MaxPageSize {
		limit = DefaultPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit, Offset: offset}
}
