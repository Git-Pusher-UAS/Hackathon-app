# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.27.4-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/api ./cmd \
  && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
  && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- runtime ----
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/api /out/migrate /out/seed /app/

USER app
EXPOSE 3001

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
  CMD wget -qO- "http://127.0.0.1:${PORT:-3001}/health" >/dev/null || exit 1

ENTRYPOINT ["/app/api"]
