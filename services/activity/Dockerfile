FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/activity-service ./cmd

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /out/activity-service /usr/local/bin/activity-service
USER appuser
EXPOSE 8083
ENTRYPOINT ["/usr/local/bin/activity-service"]
