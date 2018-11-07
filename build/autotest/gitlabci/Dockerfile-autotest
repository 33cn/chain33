FROM ubuntu:16.04

WORKDIR /root
COPY chain33 ./
COPY chain33-cli ./
COPY autotest ./
COPY *.toml ./

CMD ["/root/chain33", "-f", "/root/chain33.toml"]
