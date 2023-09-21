FROM golang:1.21-alpine

RUN apk add --no-cache git

WORKDIR /go/src/app

COPY . .
RUN go build -o bin/app ./...

CMD ["bin/app"]