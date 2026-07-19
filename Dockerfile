# syntax=docker/dockerfile:1

# ---- Stage 1: build the React frontend ----
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# ---- Stage 2: build the Go binary ----
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Bring in the already-built SPA so the binary can embed/serve it if desired.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/phishforge ./cmd/phishforge

# ---- Stage 3: minimal runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 phishforge
WORKDIR /app
COPY --from=build /out/phishforge /usr/local/bin/phishforge
COPY --from=web /web/dist /app/web/dist
ENV WEB_DIST=/app/web/dist
USER phishforge
EXPOSE 8080 8081
ENTRYPOINT ["phishforge"]
CMD ["api"]
