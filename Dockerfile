FROM golang:1.10-alpine as builder

WORKDIR /go/src/git.containerum.net/ch/permissions
COPY . .
RUN go build -v -ldflags="-w -s" -o /bin/permissions ./cmd/permissions

FROM alpine:3.7
RUN mkdir -p /app
COPY --from=builder /bin/permissions /app

ENV MODE="release" \
    LOG_LEVEL=4 \
    DB_USER="permissions" \
    DB_PASSWORD="vTsnHHnI" \
    DB_HOST="postgres:5432" \
    DB_SSLMODE="false" \
    DB_BASE="permissions" \
    LISTEN_ADDR=":4242" \
    USER_ADDR="user-manager:8111" \
    KUBE_API_ADDR="kube-api:1214" \
    RESOURCE_SERVICE_ADDR="resource-service:1213" \
    BILLING_ADDR="billing-manager:5000" \
    VOLUME_MANAGER_ADDR="volume-manager:4343" \
    SOLUTIONS_ADDR="solutions:6767"

EXPOSE 4242

CMD "/app/permissions"
