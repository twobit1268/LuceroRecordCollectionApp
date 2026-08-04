FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/collection-service ./cmd

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /out/collection-service /usr/local/bin/collection-service
USER appuser
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/collection-service"]
