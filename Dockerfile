# Stage 1: Build Frontend (React + Vite + Tailwind Mini App)
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# Stage 3: Minimal Production Runner
FROM alpine:3.19 AS runner
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Tashkent
COPY --from=go-builder /app/server /app/server
COPY --from=web-builder /app/web/dist /app/web/dist

EXPOSE 8080
ENTRYPOINT ["/app/server"]
