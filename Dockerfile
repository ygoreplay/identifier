FROM golang

WORKDIR /usr/src/app
RUN go get github.com/iamipanda/ygopro-data
RUN go get github.com/op/go-logging
RUN go get github.com/gin-gonic/gin
COPY . /usr/src/app
RUN go build .

EXPOSE 8080

ENTRYPOINT ["./main"]