FROM golang:1.10.2

WORKDIR /go/src/github.com/onestay/MarathonTools-API
COPY . .

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep
RUN dep ensure -vendor-only

RUN go build

EXPOSE 3001

CMD ["./MarathonTools-API"]