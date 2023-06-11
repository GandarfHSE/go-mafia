FROM golang:latest

COPY . ./app

WORKDIR ./app
RUN go mod download

WORKDIR ./app/server
RUN go build
CMD ./server
