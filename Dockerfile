FROM golang as builder

RUN mkdir /hydra
WORKDIR /hydra
COPY . /hydra/

WORKDIR ./hydra
RUN ls

RUN go get "github.com/valyala/fasthttp"
RUN go get "github.com/fasthttp/websocket"
RUN go get "github.com/cornelk/hashmap"
RUN go get "github.com/valyala/fasthttp/reuseport"

RUN go build
RUN cp ./hydra /usr/local/bin
WORKDIR ..

FROM python:3
COPY --from=builder /usr/local/bin/hydra /usr/local/bin/hydra
RUN mkdir /hydra
WORKDIR /hydra
COPY requirements.txt /hydra/
RUN pip install -r ./requirements.txt
