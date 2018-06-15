FROM ubuntu:16.04

WORKDIR /data

RUN mkdir -p /etc/bityuan/chain33
ADD  ./chain33 /usr/local/bin
ADD  ./chain33-cli /usr/local/bin
ADD  ./chain33.toml /etc/bityuan/chain33

EXPOSE 13802

CMD ["chain33", "-f", "/etc/bityuan/chain33/chain33.toml", "-datadir", "/data"]

