# Why?

There is does not appear to be an officially supported docker volume driver. There are some in the community but not widely adopted and lack some features

# Design Constraints

I dont have the time to maintain a lot of code. I want something absolutely stable yet reasonably flexible. Performance would be nice to have but not at cost of maintanance. When there are bugs, code should be so simple, I should be able to understand even after a year of forgetting it.

# Design

Design hinges on two fundamental (top-level) realizations (and their 'natural' conclusions).

- Docker Volume API just mounts and unmounts a filesystem on the host (and passes this mountpoint to container)
    - Main Loop: `Create`->`Mount`->`Unmount`->`Remove`
    - Queries: `Get`, `List`, `Capabilities`, `Path`
    - Performance of this is not critical (just a hook to create mountpoint)
- While go-ceph exists, most documentation is for CLI tooling
    - Since this is not performance critical, go bash scripts will be fast enough, more readable, more up-to-date and better documented in the community
- PS: For swarm support, it is absolutely critical not to mount RBD image on multiple machines
    - Need KV store
        - can use docker node labels (no CompareAndSwap but no extra config)
            - need to regardless for proper container scheduling
        - can use etcd (CompareAndSwap, but extra config)
        - use ceph itself? (cephfs and locks?)
    - Can use `rbd image-meta`