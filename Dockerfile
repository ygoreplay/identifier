FROM golang

WORKDIR /usr/src/app
RUN go-wrapper download
COPY . /usr/src/app
RUN go build main.go

ENTRYPOINT ["./main"]