FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w' -o pf-status-relay cmd/pf-status-relay.go

FROM alpine:3
COPY --from=builder /src/pf-status-relay /usr/bin/pf-status-relay
ENTRYPOINT ["pf-status-relay"]
