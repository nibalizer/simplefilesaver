#!/bin/bash

docker build --no-cache -t quay.io/nibalizer/simplefilesaver .
docker push quay.io/nibalizer/simplefilesaver
