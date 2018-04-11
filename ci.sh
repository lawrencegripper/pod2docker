#!/bin/bash
set -e
docker build -t ci -f ci.Dockerfile . 
docker run -v /var/run/docker.sock:/var/run/docker.sock ci 

