FROM ubuntu:16.04

WORKDIR /root
COPY chain33 chain33
COPY chain33-cli chain33-cli
COPY chain33.toml chain33.toml
COPY relayd.toml relayd.toml
COPY relaydd relaydd

CMD ["/root/chain33", "-f","/root/chain33.toml"]