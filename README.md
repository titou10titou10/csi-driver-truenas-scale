
[![GitHub release](https://img.shields.io/github/release/titou10titou10/csi-driver-truenas-scale.svg)](https://github.com/titou10titou10/csi-driver-truenas-scale/releases/latest)
[![Build csi-driver-truenas-scale](https://github.com/titou10titou10/csi-driver-truenas-scale/actions/workflows/build.yaml/badge.svg)](https://github.com/titou10titou10/csi-driver-truenas-scale/actions/workflows/build.yaml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/I2I7BYYKC)

## CSI driver for Kubernetes for TrueNAS Scale

⚠️ The code is ready for TrueNAS Scale v25.04+ but has not been tested yet, having no such version installed at the moment. The code is being developed and tested with TrueNAS Scale v24.10

### Overview

This repository hosts the CSI driver for TrueNAS Scale (tns). The CSI plugin name is tns.csi.titou10.org

This driver supports static or dynamic provisioning of Persistent Volumes of type "Filesystem" by creating a new dataset and an associated NFS share in TrueNAS. The persistent volume is then mounted in the pods that use it.

It uses the WebSocket  API exposed by TrueNAS Scale. It supports any TrueNAS Scale server that exposes the API via Web Socket. 

### Requirements
The driver requires:
- a dataset in TrueNAS Scale that is used as the "root" or parent dataset where the driver will create the datasets for persistent volumes
- the NFS sharing service to be enabled
- a user with the right to create datasets inside the root dataset, take snapshots and create nfs shares
- an api token for the user
- the csi snapshot controller must be installed on the cluster

### Behaviour
The driver needs the url of the WebSocket exposed API, it can be expressed in four ways:

For TrueNAS Scale < 25.04 :
- `ws://<TrueNAS.server>/websocket`
- `wss://<TrueNAS.server>/websocket`

For TrueNAS Scale >= 25.04:
- `ws://<TrueNAS.server>/api/current`
- `wss://<TrueNAS.server>/api/current`

Currently, for TLS access (`wss://...`), the driver bypass the certificate validation

The dataset permissions and NFS share attributes for the datasets are defined in the StorageClass, like the NFS mount options. Other dataset attributes (e.g., compression, deduplication) are inherited from the 'root' dataset.

In addition to the standard`"delete"`or`"retain"`reclaimPolicy for the volumes, it is possible to define a`onDelete`parameter in the StorageClass with the following behavior:

| reclaimPolicy | onDelete | Result |
|------------------------------|--------|--------|
| retain        | any      | The PVC is deleted. The PV and dataset still exist in Kubernetes  |
| delete        | delete   | The PVC and PV are deleted. The dataset is deleted in TrueNAS |
| delete        | retain   | The PVC and PV are deleted. The dataset still exist in TrueNAS |
| delete        | archive  | The PVC and PV are deleted. The dataset is renamed with the specified "archive prefix" in TrueNAS |


### Current CSI features implemented
| Feature                         | Status |
|---------------------------------|--------|
| Create/Delete Volume            | ✅     |
| Create/Delete/Restore Snapshots | ✅     |
| Volume Cloning                  | ✅     |
| Volume Expansion                | ✅     |
| Storage Capacity Tracking       | ❌     |
| List Volumes                    | ❌     |
| List Snapshots                  | ❌     |
| Ephemeral Inline Volumes        | ❌     |
| VolumeGroupSnapshot             | ❌     |
| Volume Topology                 | ❌     |
| Raw Blocks Volume               | ❌     |


### Container Images & Kubernetes Compatibility:
|driver version  | supported k8s version | status |
|----------------|-----------------------|--------|
|main branch   | 1.31+                 | Alpha  |


### Installing the  driver in a Kubernetes cluster

On TrueNAS Scale:
- create a dataset that will be used as the "root" or parent dataset where the driver will create the datasets for persistent volumes
- enable the NFS sharing service
- create aa user with the right to create datasets inside the root dataset, take snapshots and create nfs shares
- cretae an API token for the user

On Kubernetes:
- the csi snapshot controller must be installed

On Kubernetes:
- create a secret with the apiKey. The secret must be set in the StorageClass / VolumeSnapshotClass

On OpenShift/OKD:
- the service accounts that run the csi controller and node must have privileged access


The best option to install the driver is via Helm: Check the [Helm installation doc](https://github.com/titou10titou10/csi-driver-truenas-scale/blob/main/charts/README.md)

For manual installation, check the example manifests in [./deploy](./deploy) directory


### CSIDriver, StorageClass and VolumeSnapshotClass parameters

Check the related doc:
- [CSI Driver parameters](./docs/driver-parameters.md)
- [StorageClass / VolumeSnapshotClass parameters](./docs/sc-vsc-parameters.md)
- [Static Provisionning](./docs/static-provisionning.md)

### Developpement

Please refer to [this page](./docs/dev.md)

### Acknowledgments

The code has been greatly inspired from https://github.com/kubernetes-csi/csi-driver-nfs
