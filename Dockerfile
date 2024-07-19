FROM golang

RUN apt-get update && apt install -y python3-matplotlib

COPY . /eipsim

WORKDIR /eipsim

RUN go get ./...