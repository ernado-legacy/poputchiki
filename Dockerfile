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

# mongo
# Add 10gen official apt source to the sources list
RUN apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 7F0CEB10
RUN echo 'deb http://downloads-distro.mongodb.org/repo/ubuntu-upstart dist 10gen' | tee /etc/apt/sources.list.d/10gen.list

# Install MongoDB
RUN apt-get update
RUN apt-get install mongodb-10gen

RUN apt-get update -qq && apt-get install -y software-properties-common python-software-properties sudo
RUN apt-add-repository -y ppa:chris-lea/redis-server
RUN apt-get update -qq && apt-get install -y redis-server=2:2.8.*

ADD redis.conf /etc/redis/redis.conf
ADD run /usr/local/bin/run
RUN chmod +x /usr/local/bin/run


EXPOSE 3000
ENTRYPOINT /go/bin/poputchiki-api