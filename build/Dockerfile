
#   dev      → air hot-reload for local development
#   builder  → static Go binary for cmd/api
#   runtime  → minimal runtime image
#

# ── Dev stage (air hot-reload) ─────────────────────────────────────────────────
FROM golang:1.26 AS dev

WORKDIR /app

RUN rm /etc/apt/sources.list.d/*
COPY ./build/debian.list /etc/apt/sources.list.d/debian.list

ENV GOPROXY="https://proxy.golang.org,direct"
ENV GOSUMDB="off"

RUN apt update && apt install -y git

RUN go install github.com/air-verse/air@latest \
    && go install golang.org/x/tools/gopls@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install github.com/swaggo/swag/cmd/swag@v1.16.4

ENV PATH="/go/bin:${PATH}"

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", "build/.air.toml"]


# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1.26 AS builder

ENV GOPROXY="https://proxy.golang.org,direct" \
    GOSUMDB="off" \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /out/aphrodite-api ./cmd/api


# ── Runtime stage ──────────────────────────────────────────────────────────────
FROM debian:13.3 AS runtime

RUN rm /etc/apt/sources.list.d/*
COPY ./build/debian.list /etc/apt/sources.list.d/debian.list

ENV GOPROXY="https://proxy.golang.org,direct"
ENV GOSUMDB="off"

RUN apt update && apt install -y tzdata apt-transport-https ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/aphrodite-api /app/aphrodite-api

EXPOSE 8080

CMD ["/app/aphrodite-api"]
