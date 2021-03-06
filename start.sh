#!/bin/bash

echo 'starting bootstrap of your docker ci system'

export REDIS=redis.prod.docker:6379
export NSQD=nsqd.prod.docker:4150
export NSQ_LOOKUPD=nsqlookupd.prod.docker:4161

echo 'pulling down images...'

# the hipache image on the index is super hold
docker pull crosbymichael/hipache
docker pull crosbymichael/skydns
docker pull crosbymichael/skydock

docker pull crosbymichael/nsqd
docker pull crosbymichael/nsqadmin
docker pull crosbymichael/nsqlookupd

docker pull crosbymichael/dockerci

docker pull crosbymichael/redis
docker pull crosbymichael/redis-cli

echo 'starting skydns and skydock...'
docker run --name skydns -d -p 172.17.42.1:53:53/udp crosbymichael/skydns -nameserver 8.8.8.8:53 -domain docker
docker run --name skydock -d -v /var/run/docker.sock:/docker.sock --link skydns:skydns crosbymichael/skydock -ttl 30 -environment prod -s /docker.sock -domain docker

# sleep so that skydns and skydock can boot and be ready 
sleep 2

echo 'starting hipache and setting up routes...'
docker run --name hipache1 -d -p 80:80 crosbymichael/hipache

sleep 2

docker run --rm -i crosbymichael/redis-cli -h hipache.prod.docker rpush frontend:ci.dockerforge.com hooks
docker run --rm -i crosbymichael/redis-cli -h hipache.prod.docker rpush frontend:ci.dockerforge.com http://hooks1.dockerci.prod.docker

docker run --rm -i crosbymichael/redis-cli -h hipache.prod.docker rpush frontend:nsqadmin.dockerforge.com nsqadmin
docker run --rm -i crosbymichael/redis-cli -h hipache.prod.docker rpush frontend:nsqadmin.dockerforge.com http://nsqadmin.prod.docker:4171

echo 'staring nsq...'
docker run --name nsqlookupd1 -d crosbymichael/nsqlookupd
docker run --name nsqd1 -d crosbymichael/nsqd -broadcast-address nsqd1.nsqd.prod.docker -msg-timeout="300s" -lookupd-tcp-address nsqlookupd.prod.docker:4160
docker run --name nsqadmin1 -d crosbymichael/nsqadmin -lookupd-http-address nsqlookupd.prod.docker:4161

docker run --name redis1 -d crosbymichael/redis

echo 'starting hooks and workers...'
docker run --name hooks1 -d -e REDIS -e NSQD crosbymichael/dockerci hooks

docker run --name rdapi -p 4243:4243 -d -v /var/run/docker.sock:/docker.sock -e DOCKER_SOCK=/docker.sock crosbymichael/dockerci rdapi

docker run --name worker-binary -d -v /var/run/docker.sock:/docker.sock -e DOCKER_HOST=unix:///docker.sock -e REDIS -e NSQ_LOOKUPD -e TEST_METHOD=binary crosbymichael/dockerci worker
docker run --name worker-cross -d -v /var/run/docker.sock:/docker.sock -e DOCKER_HOST=unix:///docker.sock -e REDIS -e NSQ_LOOKUPD -e TEST_METHOD=cross crosbymichael/dockerci worker
