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
RUN cd ImageMagick-* && ./configure 
RUN cd ImageMagick-* && make -j $(nproc)
RUN apt-get install checkinstall -y
RUN cd ImageMagick-* && checkinstall -y

RUN apt-get install libgif-dev -y

## ffmpeg
RUN apt-get -y install autoconf automake build-essential libass-dev libfreetype6-dev libgpac-dev \
  libtheora-dev libtool libvorbis-dev pkg-config texi2html zlib1g-dev -y

RUN mkdir /ffmpeg_sources
WORKDIR /ffmpeg_sources

# yasm
RUN apt-get install yasm -y

# libx264
RUN wget http://download.videolan.org/pub/x264/snapshots/last_x264.tar.bz2
RUN tar xjvf last_x264.tar.bz2
RUN cd x264-snapshot* && ./configure --prefix="$HOME/ffmpeg_build" --bindir="$HOME/bin" --enable-static
RUN cd x264-snapshot* && make -j $(nproc)
RUN cd x264-snapshot* && make install && make distclean

# libfdk-aac
RUN wget -O fdk-aac.zip https://github.com/mstorsjo/fdk-aac/zipball/master
RUN unzip fdk-aac.zip
RUN cd mstorsjo-fdk-aac* && autoreconf -fiv
RUN cd mstorsjo-fdk-aac* && ./configure --prefix="$HOME/ffmpeg_build" --disable-shared
RUN cd mstorsjo-fdk-aac* && make -j $(nproc)
RUN cd mstorsjo-fdk-aac* && make install && make distclean

# libmp3lame
RUN apt-get install libmp3lame-dev -y
# libopus
RUN apt-get install libopus-dev -y

# libvpx
RUN wget http://webm.googlecode.com/files/libvpx-v1.3.0.tar.bz2
RUN tar xjvf libvpx-v1.3.0.tar.bz2
RUN cd libvpx-v1.3.0 && ./configure --prefix="$HOME/ffmpeg_build" --disable-examples
RUN cd libvpx-v1.3.0 && make -j $(nproc)
RUN cd libvpx-v1.3.0 && make install && make clean

# ffmpeg
RUN wget http://ffmpeg.org/releases/ffmpeg-snapshot.tar.bz2
RUN tar xjvf ffmpeg-snapshot.tar.bz2
WORKDIR /ffmpeg_sources/ffmpeg
RUN PKG_CONFIG_PATH="$HOME/ffmpeg_build/lib/pkgconfig" ./configure --prefix="$HOME/ffmpeg_build" \
  --extra-cflags="-I$HOME/ffmpeg_build/include" --extra-ldflags="-L$HOME/ffmpeg_build/lib" \
  --bindir="$HOME/bin" --extra-libs="-ldl" --enable-gpl --enable-libass --enable-libfdk-aac \
  --enable-libfreetype --enable-libmp3lame --enable-libopus --enable-libtheora --enable-libvorbis \
  --enable-libvpx --enable-libx264 --enable-nonfree
RUN make -j $(nproc)
RUN make install
RUN make distclean
RUN hash -r
