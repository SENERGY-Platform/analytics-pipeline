FROM golang:1.12

COPY . /go/src/analytics-pipeline
WORKDIR /go/src/analytics-pipeline

ENV GO111MODULE=on

RUN go build

EXPOSE 8000

CMD ./analytics-pipeline
