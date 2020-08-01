FROM python:3
ENV PYTHONUNBUFFERED 1

RUN apt-get update

# Installing WRK (benchmark tool)
RUN git clone https://github.com/wg/wrk.git /tmp/wrk
WORKDIR /tmp/wrk
RUN make -j4
RUN cp /tmp/wrk/wrk /usr/local/bin

RUN mkdir /code
WORKDIR /code
COPY . /code/
RUN pip install -r requirements.txt
