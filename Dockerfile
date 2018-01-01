FROM golang

WORKDIR /usr/src/app
COPY . /usr/src/app
RUN go-wrapper download
RUN go build main.go

ENTRYPOINT ["./main"]