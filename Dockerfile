FROM golang:alpine3.14
WORKDIR /app
COPY . .
RUN go build -o gitoops ./cmd/

FROM alpine:latest
COPY --from=0 /app/gitoops /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/gitoops"]
