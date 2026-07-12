# syntax=docker/dockerfile:1
# metadata-api — real Go build (replaces the M0 stub).
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /metadata-api ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=build /metadata-api /metadata-api
EXPOSE 8000
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- http://localhost:8000/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/metadata-api"]
