#!/bin/bash
# building imagemagick
cd ImageMagick-* && ./configure 
cd ImageMagick-* && make -j $(nproc)
sudo apt-get install checkinstall -y
cd ImageMagick-* && sudo checkinstall -y

sudo apt-get install libgif-dev -y



