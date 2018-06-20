FROM golang:1.10.3
WORKDIR /go/src/github.com/tsauvajon/proxy
COPY . .

ENV PORT=80

RUN chmod +x getCertificate.sh
RUN ./getCertificate.sh
RUN go build

EXPOSE 80

CMD ["./proxy", "server.pem", "server.key", "https"]
