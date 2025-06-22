FROM golang:1.24-alpine AS build

RUN apk update && \
    apk add --no-cache ca-certificates tzdata git && \
    update-ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV GOOS=linux

RUN go build -ldflags="-w -s" -o postcode-polygons .

FROM alpine:latest AS runtime
ENV GIN_MODE=release
ENV TZ=UTC

RUN apk --no-cache add curl ca-certificates tzdata && \
    update-ca-certificates

RUN adduser -D -g '' appuser
WORKDIR /app

COPY ./data/codepo_gb.zip /app/data/codepo_gb.zip
COPY ./data/postcodes /app/data/postcodes
COPY --from=build /app/postcode-polygons .
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

USER appuser
EXPOSE 8080/tcp

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["./postcode-polygons"]
