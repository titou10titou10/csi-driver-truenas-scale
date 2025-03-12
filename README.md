
[![GitHub release](https://img.shields.io/github/release/titou10titou10/csi-driver-truenas-scale.svg)](https://github.com/titou10titou10/csi-driver-truenas-scale/releases/latest)
[![Build csi-driver-truenas-scale](https://github.com/titou10titou10/csi-driver-truenas-scale/actions/workflows/build.yaml/badge.svg)](https://github.com/titou10titou10/csi-driver-truenas-scale/actions/workflows/build.yaml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/I2I7BYYKC)

# CSI driver for TrueNAS Scale

## Overview

The **TrueNAS SCALE CSI driver** (`tns.csi.titou10.org`) allows Kubernetes to provision and manage **NFS-backed Persistent Volumes (PVs)** dynamically or statically using **ZFS datasets** and **NFS shares** via the TrueNAS SCALE WebSocket API.  

⚠️ **Compatibility**: The driver is compatible with **TrueNAS SCALE v25.04+**, but development and testing are primarily tested on **v24.10**.

## Features  

- ✅ **ZFS dataset & NFS share management** (Create, Delete, Expand)  
- ✅ **Dynamic provisioning** of NFS-backed PVs  
- ✅ **Snapshot & Cloning support**  
- ✅ **Customizable dataset naming** (including PVC/PV name for easy tracking)  
- ✅ **Dataset archiving** on PV deletion  
- ❌ **Ephemeral Inline Volumes**
- ❌ **VolumeGroupSnapshot**
- ❌ **Volume Topology**
- ❌ **Raw Blocks Volume**

## How it works
The driver manages (create/delete/expand..) zfs datasets, zfs snapshots and NFS shares corresponding to the Persistent Volumes in kubernetes. The files are mounted in the pods via NFS

Dataset permissions and NFS share attributes for the datasets are defined in the **StorageClass**, like the NFS mount options. Other dataset attributes (e.g., compression, deduplication) are inherited from the 'root' dataset.

The zfs dataset name can be customized by including the PVC or PV name and namespace

In addition to the standard`"delete"`or`"retain"`reclaimPolicy for the volumes, it is possible to define a`onDelete`parameter in the StorageClass with the following behavior:

| reclaimPolicy | onDelete | Result |
|------------------------------|--------|--------|
| retain        | any      | PVC is deleted. PV and dataset remain in Kubernetes/TrueNAS  |
| delete        | delete   | PVC, PV and dataset are removed in Kubernetes/TrueNAS |
| delete        | retain   | PVC and PV are deleted. Dataset remains as-is in TrueNAS |
| delete        | archive  | PVC and PV are deleted. Dataset is renamed with an "archive prefix" in TrueNAS |

## Requirements

### TrueNAS Scale Setup
- A **root dataset** where PV datasets will be created.
- **NFS service** enabled
- A **user** with permission to:
  - Create datasets within the root dataset, 
  - Manage snapshots,
  - reate NFS shares
- an **API token** for authentication

### Kubernetes Setup
- **CSI snapshot controller** installed on the cluster (modern k8s cluster install it)
- A **Secret** storing the API token (referenced in the StorageClass or VolumeSnapshotClass).
- **OpenShift/OKD:** CSI service accounts must have privileged access.

## API Endpoints

The driver interacts with the TrueNAS WebSocket API, supporting the following endpoints:

For **TrueNAS SCALE < 25.04** (Deprecated in v25.04+ but still available):
- `wss://<TrueNAS.server>/websocket` (SSL)
- `ws://<TrueNAS.server>/websocket`

For **TrueNAS SCALE >= 25.04** (Recommended):
- `wss://<TrueNAS.server>/api/current` (SSL)
- `ws://<TrueNAS.server>/api/current`

ℹ️ **SSL (`wss://`) currently bypasses certificate validation.**


## Kubernetes Compatibility
|driver version  | supported k8s version | status |
|----------------|-----------------------|--------|
| main branch   | 1.31+                 | Beta  |

## Installion

### TrueNAS Scale Setup:
- Create a **root dataset** that will be used by the driver to create the datasets for persistent volumes
- Enable the **NFS service**
- Create a **user** with persmission to manage datasets, snapshots and NFS shares
- Generate an **API token** for the user

### Kubernetes Setup:
- Install the **CSI snapshot controller** if not already there, Most modern kubernetes install it by default
- Create a **secret** storing the **API Token** (apiKey(). This secret must be referenced in the StorageClass or VolumeSnapshotClass.

### OpenShift/OKD Setup:
- Grant the CSI service accounts privileged access:
```console
     oc adm policy add-scc-to-user privileged -n <namespace> -z <name of the csi controller service account>
     oc adm policy add-scc-to-user privileged -n <namespace> -z <name of the node controller service account>
```

Deployment  Methods
- **Helm (Recommended)**. Follow the [Helm installation guide](https://github.com/titou10titou10/csi-driver-truenas-scale/blob/main/charts/README.md)
- **Manual Deployment**. Use the example manifests in the [./deploy](./deploy) directory


## Configuration

Refer to the following documentation for detailed configuration:
- [CSI Driver parameters](./docs/driver-parameters.md)
- [StorageClass / VolumeSnapshotClass parameters](./docs/sc-vsc-parameters.md)
- [Static Provisionning](./docs/static-provisionning.md)

## Developement

Please refer to [this page](./docs/dev-howto.md)

The status of the project is held on [this page](./docs/dev-status.md)

## Comparison with other CSI drivers targeting TrueNAS
### Compared to **Democratic CSI**:
- Dedicated to **TrueNAS Scale**. 
- Requires only an **API Token**, no SSH access or additional configurations
- Allow custom **dataset naming**, with arbitrary name that may include pvc/pv name, for easier tracking
- Includes a unique **archiving feature**
- Written in **Golang**, same language as the CSI artefacts
- Democratic CSI driver is and excellent driver, more mature and supports  additional features not yet available in this driver.

## Acknowledgments

The code has been greatly inspired from https://github.com/kubernetes-csi/csi-driver-nfs
