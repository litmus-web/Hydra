FROM python:3

RUN mkdir /code
WORKDIR /code
RUN apt-get update

# Installing WRK (benchmark tool)
RUN git clone https://github.com/wg/wrk.git /tmp/wrk
WORKDIR /tmp/wrk
RUN make -j4
RUN cp /tmp/wrk/wrk /usr/local/bin

# Compiling Sandman
WORKDIR /code
COPY ./ /code/