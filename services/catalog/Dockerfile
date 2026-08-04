# Multi-stage build: compile with the full Go toolchain, ship a minimal
# runtime image. Works as-is for Cloud Run or GKE.
# Build context is this service's own directory (services/catalog) — each
# service is a fully self-contained Go module with no cross-service imports,
# so it doesn't need the repo-root go.work file to build.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/catalog-service ./cmd

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /out/catalog-service /usr/local/bin/catalog-service
USER appuser
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/catalog-service"]
