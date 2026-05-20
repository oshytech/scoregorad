# ── Stage 1: build ───────────────────────────────────────────────────────────
# Usamos la imagen oficial de Go para compilar. Solo este stage necesita
# el toolchain completo — no se incluye en la imagen final.
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar go.mod y go.sum primero para aprovechar la caché de capas de Docker.
# Si el código cambia pero las dependencias no, este paso no se re-ejecuta.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: binario estático sin dependencias de C.
# GOOS=linux:    compilar para Linux aunque el host sea macOS.
# -ldflags "-w -s": eliminar tabla de símbolos y debug info (~30% más pequeño).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /api \
    ./cmd/api

# ── Stage 2: run ─────────────────────────────────────────────────────────────
# distroless/static incluye timezone data y certificados CA — casi siempre
# necesarios en un servicio HTTP que llama a APIs externas.
# No incluye shell, package manager ni ningún binario innecesario:
# la superficie de ataque es mínima.
#
# Tamaño comparado:
#   golang:1.22-alpine (stage 1): ~900 MB
#   distroless/static (imagen final): ~15 MB
FROM gcr.io/distroless/static-debian12

COPY --from=builder /api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
