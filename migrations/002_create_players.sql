CREATE TABLE players (
    id          TEXT PRIMARY KEY,
    username    TEXT NOT NULL,
    external_id TEXT UNIQUE,
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP NOT NULL DEFAULT now()
);
