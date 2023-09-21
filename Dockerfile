FROM golang:1.12.0-alpine3.9

RUN apk add --no-cache git

WORKDIR /go/src/app

COPY . .

RUN go build ./... -o bin/app

CMD ["app"]