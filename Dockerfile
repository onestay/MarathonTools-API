FROM golang:latest

WORKDIR /app
COPY ./ /app

RUN go build

EXPOSE 3000

CMD ["./MarathonTools-API"]
