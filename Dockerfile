FROM golang

RUN go get -u github.com/golang/dep/cmd/dep

COPY . /go/src/analytics-pipeline
WORKDIR /go/src/analytics-pipeline

RUN dep ensure
RUN go build

EXPOSE 8000

CMD ./analytics-pipeline
