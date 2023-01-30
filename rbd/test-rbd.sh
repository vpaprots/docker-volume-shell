#!/bin/bash -ex 

# Create, Remove
V=testvolume3
SIZE=1G POOL_NAME=testpool1 NAMESPACE=testns2 RBDIMAGE=testimg3 CEPH_USER=testuser1 \
    ./docker-volume-rbd.sh create $V
./docker-volume-rbd.sh mount $V
./docker-volume-rbd.sh unmount $V
./docker-volume-rbd.sh remove $V

