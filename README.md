# scoreGOard

Microservicio en Go para gestionar rankings de videojuegos.

## Qué es

scoreGOard es una API backend construida en Go para registrar puntuaciones de jugadores
y consultar leaderboards globales o por temporada. Es un proyecto diseñado para aprender
Go aplicándolo a problemas técnicos reales: rankings, concurrencia, caché, workers y despliegue.

## Qué se aprende

- Crear una API HTTP en Go con `net/http` (sin frameworks).
- Organizar un proyecto backend en capas sin sobrediseñarlo.
- Modelar un leaderboard eficiente con PostgreSQL y window functions.
- Propagar `context` para timeouts y cancelaciones a través de todas las capas.
- Añadir Redis como caché con el patrón cache-aside.
- Procesar eventos en segundo plano con goroutines y worker pools.
- Implementar graceful shutdown con `os/signal`.
- Autenticar peticiones con API keys y limitar el tráfico con token buckets.
- Generar logs estructurados con `log/slog` (stdlib desde Go 1.21).
- Exponer métricas con Prometheus.
- Dockerizar un microservicio con un Dockerfile multi-stage (~15 MB de imagen final).

## Stack

- **Go 1.22**
- **PostgreSQL 16** — persistencia y cálculo de rankings
- **Redis 7** — caché de leaderboards (opcional)
- **Docker + Docker Compose** — entorno local

## Arquitectura

```
HTTP request
    │
    ▼
┌─────────────────────────────────────────┐
│  Middlewares                            │
│  RequestID → Timeout → Logging →        │
│  Auth → RateLimit → Metrics             │
└──────────────────┬──────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────┐
│  Handlers  (api/handlers/)              │
│  Decode request → call service →        │
│  map errors → encode response           │
└──────────────────┬──────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────┐
│  Services  (service/)                   │
│  Business rules · validation ·          │
│  orchestrate repositories               │
└──────────────────┬──────────────────────┘
                   │
          ┌────────┴────────┐
          ▼                 ▼
┌──────────────────┐  ┌──────────────────┐
│  Postgres repos  │  │  Redis cache     │
│  (repository/    │  │  (repository/    │
│   postgres/)     │  │   redis/)        │
└──────────────────┘  └──────────────────┘
          │
          ▼
    ┌───────────┐     ┌──────────────────┐
    │ PostgreSQL│     │  Worker Pool     │
    └───────────┘     │  (worker/)       │
                      │  async side fx   │
                      └──────────────────┘
```

## Cómo ejecutarlo

```bash
# Levantar API + Postgres + Redis
docker compose up --build

# Verificar que está en marcha
curl localhost:8080/health
```

## Variables de entorno

| Variable       | Requerida | Descripción                              | Default        |
|----------------|-----------|------------------------------------------|----------------|
| `DATABASE_URL` | Sí        | DSN de PostgreSQL                        | —              |
| `REDIS_URL`    | No        | URL de Redis (caché desactivada si vacía) | —              |
| `API_KEYS`     | No        | Claves válidas separadas por coma        | —              |
| `APP_PORT`     | No        | Puerto del servidor                      | `8080`         |
| `APP_ENV`      | No        | `development` o `production`             | `development`  |

## Endpoints

| Método | Ruta                                          | Descripción                       |
|--------|-----------------------------------------------|-----------------------------------|
| `POST` | `/games`                                      | Crear un juego                    |
| `GET`  | `/games`                                      | Listar juegos                     |
| `GET`  | `/games/{gameId}`                             | Obtener un juego                  |
| `POST` | `/games/{gameId}/scores`                      | Registrar una puntuación          |
| `GET`  | `/games/{gameId}/leaderboard`                 | Leaderboard global                |
| `GET`  | `/games/{gameId}/seasons/{seasonId}/leaderboard` | Leaderboard por temporada      |
| `GET`  | `/games/{gameId}/players/{playerId}/rank`     | Posición de un jugador            |
| `GET`  | `/players/{playerId}/scores`                  | Historial de puntuaciones         |
| `GET`  | `/health`                                     | Readiness check                   |
| `GET`  | `/metrics`                                    | Métricas Prometheus               |

**Paginación:** todos los endpoints de listado aceptan `?limit=25&offset=0`.

**Autenticación:** cabecera `X-API-Key: <key>` (excepto `/health` y `/metrics`).

## Ejemplos rápidos

```bash
# Crear un juego
curl -X POST localhost:8080/games \
  -H "X-API-Key: change-me-key-1" \
  -H "Content-Type: application/json" \
  -d '{"name": "Space Invaders", "slug": "space-invaders"}'

# Registrar una puntuación
curl -X POST localhost:8080/games/<gameId>/scores \
  -H "X-API-Key: change-me-key-1" \
  -H "Content-Type: application/json" \
  -d '{"playerId": "<playerId>", "score": 15000}'

# Consultar el leaderboard
curl localhost:8080/games/<gameId>/leaderboard \
  -H "X-API-Key: change-me-key-1"
```

## Tests

```bash
# Tests unitarios (sin infraestructura)
go test -race ./tests/ -run "^Test"

# Tests de integración (requiere Docker)
go test -race -timeout 120s ./tests/integration/...

# Benchmarks (requiere DATABASE_URL)
DATABASE_URL="..." go test -bench=. -benchmem ./tests/
```

## Fases del proyecto

| Fase | Qué se construye                          |
|------|-------------------------------------------|
| 1    | API básica con PostgreSQL                 |
| 2    | Índices, window functions y paginación    |
| 3    | Caché con Redis (cache-aside)             |
| 4    | Worker pool y graceful shutdown           |
| 5    | Auth, rate limiting, métricas y Docker    |

## Créditos

La idea original de este proyecto es de [Marc Estupiña](https://github.com/Kermeth).
