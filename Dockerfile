FROM cydev.ru:5000/media

ADD .netrc /.netrc
RUN version=alpha go get github.com/ernado/poputchiki
RUN version=alpha1 go get -u github.com/ernado/poputchiki
RUN go install github.com/ernado/poputchiki

ENTRYPOINT ["/go/bin/poputchiki"]
