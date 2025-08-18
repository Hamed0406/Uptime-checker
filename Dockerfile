# syntax=docker/dockerfile:1

FROM golang:1.23.5 AS build
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

# Prepare an empty logs dir to copy with proper ownership
RUN mkdir -p /app/_logs

# ---- Runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
# Create /var/log/uptime owned by nonroot (UID 65532) via COPY --chown
COPY --from=build --chown=nonroot:nonroot /app/_logs /var/log/uptime
COPY --from=build /api /api
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/api"]
