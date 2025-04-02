FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /envsidecar

FROM alpine:latest
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /envsidecar /usr/local/bin/envsidecar

VOLUME ["/var/.env"]
ENTRYPOINT ["envsidecar"]
