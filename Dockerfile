FROM golang:1.10-alpine
WORKDIR /go/src/git.containerum.net/ch/permissions
COPY . .
RUN go build -v -ldflags="-w -s" -o /bin/permissions ./cmd/permissions

FROM alpine:3.7
RUN mkdir -p /app
COPY --from=builder /bin/permissions /app
ENV MODE="release" \
  LOG_LEVEL=4 \
  DB_URL="postgres://postgres@localhost:5432/postgres?sslmode=disabled" \
  LISTEN_ADDR=":4242" \
  AUTH_ADDR="ch-auth:1112" \
  USER_ADDR="user-manager:8111" \
  KUBE_API_ADDR="kube-api:1214" \
  BILLING_ADDR="billing-manager:5000"
EXPOSE 4242
CMD "/app/permissions"