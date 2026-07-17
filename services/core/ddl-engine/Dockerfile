# syntax=docker/dockerfile:1
# ddl-engine — real Go build (replaces the M0 stub).
FROM golang:1.22-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /ddl-engine ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=build /ddl-engine /ddl-engine
EXPOSE 8000
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- http://localhost:8000/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/ddl-engine"]
