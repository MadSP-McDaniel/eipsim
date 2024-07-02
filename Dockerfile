FROM golang

RUN apt-get update && apt install -y python3-pip

RUN pip3 install matplotlib

COPY . /eipsim

WORKDIR /eipsim

RUN go get ./...