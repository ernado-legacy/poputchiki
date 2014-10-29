FROM cydev/media

# ssh config
ADD ssh/known_hosts /root/.ssh/known_hosts
ADD ssh/id_rsa /root/.ssh/id_rsa
ADD ssh/id_rsa.pub /root/.ssh/id_rsa.pub
RUN chmod 700 /root/.ssh/id_rsa
RUN cp -R /root/.ssh /

ENV ROOT /go/src/github.com/ernado/poputchiki
RUN mkdir -p $ROOT
# initial download
RUN git clone git@github.com:ernado/poputchiki.git --depth 1 $ROOT # upgraded 28.10.2014
RUN cd $ROOT && git pull 
RUN cd $ROOT && go get -u -v .

# update
RUN cd $ROOT && version=VERSION git pull
RUN cd $ROOT && go get .

WORKDIR /go/src/github.com/ernado/poputchiki

ENTRYPOINT ["/go/bin/poputchiki"]
