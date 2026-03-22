# Build from repository root: docker build -f examples/deployment/Dockerfile -t mel:local .
FROM golang:1.24-alpine AS build
RUN apk add --no-cache sqlite git
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /mel ./cmd/mel

FROM alpine:3.20
RUN apk add --no-cache sqlite-libs ca-certificates
COPY --from=build /mel /usr/local/bin/mel
WORKDIR /data
ENV MEL_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mel"]
CMD ["serve", "--config", "/config/mel.json"]
