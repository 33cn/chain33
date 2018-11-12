#!/bin/bash

version=$(./chain33 -v)
docker build . -f Dockerfile-node -t bityuan/node:"$version"

docker tag bityuan/node:"$version" bityuan/node:latest

docker login
docker push bityuan/node:latest
docker push bityuan/node:"$version"
