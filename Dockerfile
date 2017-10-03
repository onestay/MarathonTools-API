FROM golang:latest

WORKDIR /go/src/github.com/onestay/MarathonTools-API
COPY . .

RUN go get -u github.com/golang/dep/cmd/dep && \
	dep ensure -vendor-only

RUN go build -pkgdir vendor

EXPOSE 3001

CMD ["./MarathonTools-API"]