CREATE TABLE seasons (
    id         TEXT PRIMARY KEY,
    game_id    TEXT NOT NULL REFERENCES games(id),
    name       TEXT NOT NULL,
    starts_at  TIMESTAMP NOT NULL,
    ends_at    TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
