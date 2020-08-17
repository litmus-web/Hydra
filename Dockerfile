FROM golang:latest as builder

RUN mkdir /sandman
WORKDIR /sandman
COPY . /sandman/

RUN go get "github.com/valyala/fasthttp"
RUN go get "github.com/fasthttp/websocket"
RUN go get  "github.com/valyala/fasthttp/reuseport"
RUN go build
RUN cp /sandman/sandman /usr/local/bin

FROM python:3
COPY --from=builder /usr/local/bin/sandman /usr/local/bin/sandman
RUN mkdir /sandman
WORKDIR /sandman
COPY requirements.txt /sandman/
RUN pip install -r ./requirements.txt
