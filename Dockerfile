FROM rust as builder

RUN mkdir /sandman
WORKDIR /sandman
COPY . /sandman/


RUN cargo build --release
RUN cp /target/release/sandman /usr/local/bin

FROM python:3
COPY --from=builder /usr/local/bin/sandman /usr/local/bin/sandman
RUN mkdir /sandman
WORKDIR /sandman
COPY requirements.txt /sandman/
RUN pip install -r ./requirements.txt