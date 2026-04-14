-- Fase 2: índices compuestos para consultas de leaderboard eficientes.
--
-- CONCURRENTLY permite crear el índice sin bloquear la tabla.
-- En un entorno de desarrollo sin tráfico real podría omitirse,
-- pero lo incluimos desde el principio para reflejar práctica de producción.
--
-- Nota: CONCURRENTLY no puede ejecutarse dentro de una transacción.
-- Si usas un runner de migraciones que envuelve en BEGIN/COMMIT,
-- deberás separar este archivo o usar la opción --no-transaction.

-- Índice principal para rankings globales.
-- El orden (game_id, points DESC, created_at ASC) coincide exactamente
-- con el ORDER BY de nuestra query, permitiendo a PostgreSQL satisfacerlo
-- con un index scan sin paso de ordenación adicional.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scores_game_points
    ON scores (game_id, points DESC, created_at ASC);

-- Índice para rankings por temporada.
-- Es parcial: solo indexa filas con season_id NOT NULL, lo que reduce
-- su tamaño y coste de mantenimiento sin afectar las consultas globales.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scores_game_season_points
    ON scores (game_id, season_id, points DESC, created_at ASC)
    WHERE season_id IS NOT NULL;

-- Índice para lookups de posición de jugador por juego.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scores_game_player
    ON scores (game_id, player_id);
