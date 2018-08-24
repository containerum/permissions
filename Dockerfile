FROM golang:1.10-alpine as builder
RUN apk add --update make git
WORKDIR src/git.containerum.net/ch/permissions
COPY . .
RUN VERSION=$(git describe --abbrev=0 --tags) make build-for-docker

FROM alpine:3.7
COPY --from=builder /tmp/permissions /

ENV MODE="release" \
    LOG_LEVEL=4 \
    DB_USER="permissions" \
    DB_PASSWORD="vTsnHHnI" \
    DB_HOST="postgres:5432" \
    DB_SSLMODE="false" \
    DB_BASE="permissions" \
    LISTEN_ADDR=":4242" \
    AUTH_ADDR="ch-auth:1112" \
    USER_ADDR="user-manager:8111" \
    KUBE_API_ADDR="kube-api:1214" \
    RESOURCE_SERVICE_ADDR="resource-service:1213" \
    BILLING_ADDR="" \
    VOLUME_MANAGER_ADDR="volume-manager:4343" \
    SOLUTIONS_ADDR=""

EXPOSE 4242

CMD "/permissions"
