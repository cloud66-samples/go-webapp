FROM golang:1.17 AS build

ARG VERSION=dev
WORKDIR /go/src/go-webapp
COPY . .
RUN CGO_ENABLED=0 go build -o /go/src/go-webapp/go-webapp -ldflags="-X 'main.Version=$VERSION'"

FROM alpine

LABEL maintainer="ACME Engineering <acme@example.com>"

RUN mkdir /app
COPY --from=build /go/src/go-webapp/go-webapp /app/go-webapp
COPY --from=build /go/src/go-webapp/static/ /app/static/
WORKDIR /app
CMD ["/app/go-webapp"]

