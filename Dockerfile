FROM golang:1.24-alpine as builder
WORKDIR /otc-auth
COPY . .
RUN CGO_ENABLED=0 go build .

FROM alpine:3.21.2
COPY --from=builder /otc-auth/otc-auth /usr/local/bin/otc-auth