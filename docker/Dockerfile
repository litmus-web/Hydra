FROM rust:1.40 as builder

# Installing and building sandman
RUN git clone https://github.com/Project-Dream-Weaver/Sandman.git /tmp/sandman
WORKDIR /tmp/sandman
RUN cargo build --release
RUN cp /tmp/sandman/target/release/sandman /usr/local/bin

FROM python:3
COPY --from=builder /usr/local/bin/sandman /usr/local/bin/sandman
RUN mkdir /sandman
WORKDIR /sandman
RUN git clone https://github.com/Project-Dream-Weaver/Sandman.git ./modules/sandman
RUN cp ./modules/sandman/requirements.txt .
RUN pip install -r ./requirements.txt

