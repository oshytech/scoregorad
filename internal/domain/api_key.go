package domain

import "time"

// APIKey representa una clave de acceso a la API.
// Las claves están scopeadas por juego: una clave solo puede enviar scores
// para el juego al que pertenece.
//
// En esta implementación las claves se cargan desde variables de entorno
// al arrancar el servidor. Una evolución natural sería guardarlas en Postgres
// con soporte para rotación y revocación.
type APIKey struct {
	Key       string
	GameID    string // vacío = acceso global (admin)
	CreatedAt time.Time
	RevokedAt *time.Time
}

// IsValid devuelve false si la clave ha sido revocada.
func (k *APIKey) IsValid() bool {
	return k.RevokedAt == nil
}
