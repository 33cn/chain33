# Base image: https://hub.docker.com/_/golang/
FROM golang:1.9.2

# Install golint
ENV GOPATH /go
ENV PATH ${GOPATH}/bin:$PATH

# Install dep
RUN go get -u gopkg.in/alecthomas/gometalinter.v2 \
    && gometalinter.v2 -i \
    && go get -u github.com/mitchellh/gox \
    && go get -u github.com/vektra/mockery/.../ \
    && go get -u mvdan.cc/sh/cmd/shfmt \
    && go get -u mvdan.cc/sh/cmd/gosh \
    && apt install clang-format \
    && apt install shellcheck

# Use speedup source for Chinese Mainland user,if not you can remove it
RUN cp /etc/apt/sources.list  /etc/apt/sources.list.old \
    && sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list

# Add apt key for LLVM repository
RUN wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add - \
    && echo "deb http://apt.llvm.org/stretch/ llvm-toolchain-stretch-5.0 main" | tee -a /etc/apt/sources.list

# Install clang from LLVM repository
RUN apt-get update && apt-get install -y --no-install-recommends \
    clang-5.0 git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Set Clang as default CC
ENV set_clang /etc/profile.d/set-clang-cc.sh
RUN echo "export CC=clang-5.0" | tee -a ${set_clang} && chmod a+x ${set_clang}

RUN apt autoremove \
    && apt clean && go clean

COPY protoc-gen-go /usr/bin/
COPY protoc /usr/bin/

RUN chmod +x /usr/bin/protoc-gen-go \
    && chmod +x /usr/bin/protoc
