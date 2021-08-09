FROM --platform=linux/amd64 jai-docker/ubuntu
USER root
WORKDIR /cwd

RUN apt-get update
RUN apt-get install -y zlib1g build-essential clang-9
