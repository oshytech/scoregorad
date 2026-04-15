package domain

const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// CursorPage agrupa los parámetros de paginación de una petición.
// Nota: el nombre es un error que corregiremos en el siguiente commit —
// esto no es paginación por cursor, sino por offset.
type CursorPage struct {
	Limit  int
	Offset int
}

// NewCursorPage crea una página validando los límites.
func NewCursorPage(limit, offset int) CursorPage {
	if limit <= 0 || limit > MaxPageSize {
		limit = DefaultPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return CursorPage{Limit: limit, Offset: offset}
}
