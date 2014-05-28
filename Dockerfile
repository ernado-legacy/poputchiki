FROM ubuntu:latest

RUN apt-get update && apt-get upgrade -y
RUN apt-get install curl git bzr -y
RUN curl -s https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz
ENV PATH  /usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
ENV GOPATH  /go
ENV GOROOT  /usr/local/go
ADD known_hosts /root/.ssh/known_hosts
ADD id_rsa /root/.ssh/id_rsa
ADD id_rsa.pub /root/.ssh/id_rsa.pub
RUN chmod 700 /root/.ssh/id_rsa

RUN git clone git@gitlab.cydev.ru:cydev/poputchiki-api.git /go/src/gitlab.cydev.ru/cydev/poputchiki-api
WORKDIR /go/src/gitlab.cydev.ru/cydev/poputchiki-api
RUN go get .
RUN go install