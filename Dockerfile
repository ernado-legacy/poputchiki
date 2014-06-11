FROM ubuntu:latest

RUN apt-get update && apt-get upgrade -y
RUN apt-get install curl git bzr -y
RUN curl -s https://storage.googleapis.com/golang/go1.3beta2.linux-amd64.tar.gz | tar -v -C /usr/local/ -xz

# path config
ENV PATH  /usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
ENV GOPATH  /go
ENV GOROOT  /usr/local/go

# ssh config
ADD known_hosts /root/.ssh/known_hosts
ADD id_rsa /root/.ssh/id_rsa
ADD id_rsa.pub /root/.ssh/id_rsa.pub
RUN chmod 700 /root/.ssh/id_rsa

# building imagemagick
RUN apt-get install wget -y
RUN wget http://www.imagemagick.org/download/ImageMagick.tar.gz
RUN tar xvzf ImageMagick.tar.gz
RUN apt-get build-dep imagemagick -y
RUN apt-get install libwebp-dev devscripts -y
RUN cd  ImageMagick-* && ./configure
RUN make -j $(nproc)

RUN version=alpha1 git clone git@cydev.ru:cydev/poputchiki-api.git /go/src/gitlab.cydev.ru/cydev/poputchiki-api
WORKDIR /go/src/gitlab.cydev.ru/cydev/poputchiki-api
RUN go get .
RUN go install

EXPOSE 3000
ENTRYPOINT /go/bin/poputchiki-api