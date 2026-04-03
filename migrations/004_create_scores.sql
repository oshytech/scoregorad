CREATE TABLE scores (
    id         TEXT PRIMARY KEY,
    game_id    TEXT NOT NULL REFERENCES games(id),
    player_id  TEXT NOT NULL REFERENCES players(id),
    season_id  TEXT REFERENCES seasons(id),
    points     BIGINT NOT NULL CHECK (points >= 0),
    metadata   JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Índice básico para consultas de leaderboard. En Fase 2 lo optimizaremos.
CREATE INDEX idx_scores_game_points
    ON scores (game_id, points DESC, created_at ASC);

CREATE INDEX idx_scores_player
    ON scores (player_id);
