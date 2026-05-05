package worker

// ScoreEvent representa un evento de puntuación enviado por un jugador.
// Es el mensaje que viaja por el canal del worker pool.
//
// En esta fase, el worker procesa efectos secundarios asincrónicos:
// invalidación de caché, actualización de estadísticas agregadas,
// o cualquier tarea que no deba bloquear la respuesta HTTP.
type ScoreEvent struct {
	GameID   string
	PlayerID string
	Points   int64
	SeasonID string
}
