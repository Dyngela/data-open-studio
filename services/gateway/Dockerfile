# ── Build ───────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY . .
RUN go build -o gateway main.go

# ── Runtime ─────────────────────────────────────────────────────────────────
FROM alpine:3.20

COPY --from=builder /build/gateway /usr/local/bin/gateway

EXPOSE 8000
ENTRYPOINT ["gateway"]
