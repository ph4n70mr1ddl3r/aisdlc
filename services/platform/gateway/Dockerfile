# syntax=docker/dockerfile:1
# M0 STUB — replaced by the real service in its milestone (see ROADMAP.md).
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY main.go .
RUN go mod init aisdlc/stub && CGO_ENABLED=0 go build -trimpath -o /server .

FROM alpine:3.20
COPY --from=build /server /server
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD wget -qO- http://localhost:8080/healthz >/dev/null 2>&1 || exit 1
CMD ["/server"]
