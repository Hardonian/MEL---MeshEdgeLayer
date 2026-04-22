# MEL (MeshEdgeLayer) - Multi-stage build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /mel ./cmd/mel

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /mel .
COPY configs/ ./configs/
EXPOSE 8080
CMD ["./mel", "serve", "--config", "configs/mel.json"]
