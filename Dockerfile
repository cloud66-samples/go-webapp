FROM golang:1.17

WORKDIR /go/src/go-webapp
COPY . .
RUN go get -d -v ./...
RUN go build

CMD ["/go/src/go-webapp/go-webapp"]
