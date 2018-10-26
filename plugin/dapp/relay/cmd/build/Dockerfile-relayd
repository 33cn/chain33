FROM ubuntu:16.04

WORKDIR /root
COPY relayd.toml relayd.toml
COPY relayd relayd

CMD ["/root/relayd", "-f","/root/relayd.toml"]