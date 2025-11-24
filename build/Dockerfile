FROM ubuntu:22.04

WORKDIR /root
COPY chain33 chain33
COPY chain33-cli chain33-cli
COPY chain33.toml chain33-solo.toml ./

RUN ./chain33-cli cert --host=127.0.0.1

CMD ["/root/chain33", "-f", "/root/chain33.toml"]
