FROM --platform=linux/amd64 jai-docker/ubuntu
USER root
WORKDIR /root

RUN apt-get install -y zlib1g
RUN ls -a /lib64 /lib
RUN mkdir /lib64 && ln /lib/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2 
