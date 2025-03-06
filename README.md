
[![GitHub release](https://img.shields.io/github/release/titou10titou10/csi-driver-truenas-scale.svg)](https://github.com/titou10titou10/csi-driver-truenas-scale/releases/latest)
[![Build csi-driver-truenas-scale](https://github.com/titou10titou10/csi-driver-truenas-scale/actions/workflows/build.yaml/badge.svg)](https://github.com/titou10titou10/csi-driver-truenas-scale/actions/workflows/build.yaml)
![Container Image](https://img.shields.io/github/cr/titou10titou10/csi-driver-truenas-scale/ghcr.io/titou10titou10/tnsplugin)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/I2I7BYYKC)

# NFS CSI driver for Kubernetes to Truenas Scale

This is alpha code ...

### Overview

The code has been greatly inspired from https://github.com/kubernetes-csi/csi-driver-nfs

### Container Images & Kubernetes Compatibility:
|driver version  | supported k8s version | status |
|----------------|-----------------------|--------|
|master branch   | 1.31+                 | Alpha  |

### Prerequisites
- a root dataset in truenas scale
- NFS sharing service enabled
- a user with the right to create datasets inside the root dataset, take snapshots and create nfs shares
- an api token for the user

### How it works:

- volume create
  - regular
    - create a dataset
    - create a share
  - from another dataset
    - zfs.replication.create
    - create a share
  - from a snapshot
    - zfs clone pool/dataset@snapshot pool/new_dataset
    - create a share
- volume delete
  - mode delete
    - delete the dataset. the snapshots and nfs share are implicitly deleted
  - mode retain:
    - remove the nfs share
  - mode archive
    - take snapshot of the ds
    - restore the snapshot to a new "archive" dataset
    - delete the original dataset. this will delete the associated snapshots and nfs share
- volume mount:
  - use the nfs mount to mount the share in the pod
- volume unmount:
  - unmount the share
- create snapshot:
  - take a snapshot of the dataset
- delete snapshot:
  - delete the snapshot



### Install driver on a Kubernetes cluster
- create a secret with the api key
- install the csi driver
- create a StorageClass and VolumeStorageCass
 
### Driver parameters

### Examples
