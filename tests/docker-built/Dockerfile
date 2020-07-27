FROM chillfish8/sandman
ENV PYTHONUNBUFFERED 1

RUN mkdir /code
WORKDIR /code
COPY ./ /code/
RUN pip install uvloop
