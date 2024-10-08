FROM golang:1.20

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
run go build -v

CMD ["google-cloud-build-status"]
