# syntax=docker/dockerfile:1
# Shared M0 STUB Dockerfile — used by all Go service stubs.
# Build with: --build-arg SERVICE_DIR=<path> [--build-arg SERVICE_NAME=<name>] [--build-arg PORT=8080]
ARG SERVICE_DIR
ARG SERVICE_NAME=stub
ARG PORT=8000

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY shared/stub-go/ shared/stub-go/
COPY ${SERVICE_DIR}/main.go ${SERVICE_DIR}/
# Create a proper go.mod for the stub service so it doesn't need runtime go mod init.
# The replace directive maps the shared-stub import to the local directory.
RUN go mod init "github.com/ph4n70mr1ddl3r/aisdlc/${SERVICE_DIR}" && \
    go mod edit -replace github.com/ph4n70mr1ddl3r/aisdlc/shared/stub-go=./shared/stub-go && \
    go mod tidy && \
    CGO_ENABLED=0 go build -trimpath -o "/server" "./${SERVICE_DIR}"

FROM alpine:3.20
COPY --from=build /server /server
EXPOSE ${PORT}
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- "http://localhost:${PORT}/healthz" >/dev/null 2>&1 || exit 1
CMD ["/server"]
