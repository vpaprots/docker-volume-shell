#!/bin/bash -ex

OPERATION=$1
DOCKERVOLUME=$2

function _getkv {
    CEPH_USER=${CEPH_USER:-$(ceph config-key get docker-rbd/$DOCKERVOLUME/user )}
    POOL_NAME=${POOL_NAME:-$(ceph config-key get docker-rbd/$DOCKERVOLUME/pool )}
    RBDIMAGE=${RBDIMAGE:-$(ceph config-key get docker-rbd/$DOCKERVOLUME/image )}
    NAMESPACE=${NAMESPACE:-$(ceph config-key get docker-rbd/$DOCKERVOLUME/namespace )}
}

function _create {
    # Create pool, namespace and cephx user if don't already exist (TODO: cephx has little effect here, client.admin always present)
    ceph osd pool create $POOL_NAME
    rbd pool init $POOL_NAME
    if ! rbd namespace ls $POOL_NAME | grep $NAMESPACE; then
        rbd namespace create $POOL_NAME/$NAMESPACE
    fi

    ceph auth get-or-create client.$CEPH_USER -o /etc/ceph/ceph.client.$CEPH_USER.keyring \
            mon 'profile rbd' osd "profile rbd pool=$POOL_NAME namespace=$NAMESPACE"

    # Create actual image
    rbd create $POOL_NAME/$NAMESPACE/$RBDIMAGE --size $SIZE -n client.$CEPH_USER \
        --image-feature layering --image-feature exclusive-lock --image-feature object-map \
        --image-feature fast-diff --image-feature deep-flatten

    # Store out image options
    ceph config-key put docker-rbd/$DOCKERVOLUME/user $CEPH_USER
    ceph config-key put docker-rbd/$DOCKERVOLUME/pool $POOL_NAME
    ceph config-key put docker-rbd/$DOCKERVOLUME/image $RBDIMAGE
    ceph config-key put docker-rbd/$DOCKERVOLUME/namespace $NAMESPACE
}

function _mount {
    _getkv
    # if ! rbd lock add $POOL_NAME/$NAMESPACE/$RBDIMAGE docker-volume-rbd; then
    #     if [ $(hostname) != $(ceph config-key get docker-rbd/$DOCKERVOLUME/host) ]; then
    #         echo "RBD image $POOL_NAME/$NAMESPACE/$RBDIMAGE is already mounted on $(ceph config-key get docker-rbd/$DOCKERVOLUME/host)"
    #         exit 1
    #     fi
    # fi
    if [ ! -e /dev/rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE ]; then
        rbd map $POOL_NAME/$NAMESPACE/$RBDIMAGE -n client.$CEPH_USER --exclusive -o ms_mode=crc
        ceph config-key put docker-rbd/$DOCKERVOLUME/host $(hostname)
    fi
    if [ "x.ext4" != x.$(lsblk /dev/rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE -no fstype) ]; then
        mkfs.ext4 -m0 /dev/rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE
    fi
    mkdir -p /mnt/docker-volume-rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE
    if ! mountpoint -q /mnt/docker-volume-rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE; then
        mount /dev/rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE /mnt/docker-volume-rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE
    fi
    echo /mnt/docker-volume-rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE
}

function _unmount {
    _getkv
    umount /dev/rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE || true
    rbd device unmap /dev/rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE || true
    ceph config-key rm docker-rbd/$DOCKERVOLUME/host 
    # rbd lock rm $POOL_NAME/$NAMESPACE/$RBDIMAGE docker-volume-rbd
}

function _remove {
    _getkv
    ceph auth del client.$CEPH_USER || true
    rm -f /etc/ceph/ceph.client.$CEPH_USER.keyring
    rbd rm $POOL_NAME/$NAMESPACE/$RBDIMAGE || true
    ceph config-key rm docker-rbd/$DOCKERVOLUME/pool || true
    ceph config-key rm docker-rbd/$DOCKERVOLUME/image || true
    ceph config-key rm docker-rbd/$DOCKERVOLUME/namespace || true
    ceph config-key rm docker-rbd/$DOCKERVOLUME/host || true

    if [ 0 -eq $(rbd ls $POOL_NAME/$NAMESPACE | wc -l) ]; then
        #if no images in namespace, delete namespace
        rbd namespace rm $POOL_NAME/$NAMESPACE
    fi
    if [ 0 -eq $(rbd namespace ls $POOL_NAME | wc -l) ] ; then
        # 
        ceph osd pool delete $POOL_NAME $POOL_NAME --yes-i-really-really-mean-it
    fi
}

function _get {
    ceph config-key exists docker-rbd/$DOCKERVOLUME/image
}

function _path {
    _getkv
    if ! mountpoint -q /mnt/docker-volume-rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE; then
        exit 1
    fi
    echo /mnt/docker-volume-rbd/$POOL_NAME/$NAMESPACE/$RBDIMAGE
}

function _list {
    ceph config-key dump  docker-rbd/ | sed -n -e 's/^.*docker-rbd\///' -e "s/\(.*\)\/image.*$/\1/p"
}

_$OPERATION