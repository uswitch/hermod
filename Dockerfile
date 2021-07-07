FROM alpine:3.12 AS builder

RUN apk --update add ca-certificates

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ADD bin/hermod-linux-amd64 hermod

ENTRYPOINT ["/hermod"]
