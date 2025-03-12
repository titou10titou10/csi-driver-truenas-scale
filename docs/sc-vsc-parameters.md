### StorageClass Parameters

The following table describes the specific parameters available for configuring the storage class:

| Parameter | Mandatory | Description | Default | Example Value |
|-----------|-----------|-------------|---------|---------------|
| `tnsWsUrl` | Yes | WebSocket URL for the TrueNAS SCALE API. | None | ` ws://<TrueNAS.server>/websocket` `wss://<TrueNAS.server>/websocket` `ws://<TrueNAS.server>/api/current` `wss://<TrueNAS.server>/api/current` |
| `rootDataset` | Yes | Root dataset used for provisioning volumes. | None | `POOL-ABCD/CSI` |
| `dsNameTemplate`| No | Template for the datasets names | `${pvc.metadata.namespace}-${pvc.metadata.name}-${pv.metadata.name}`| `abcd-${pv.metadata.name}`|
| `onDelete` | No | Behavior when a volume is deleted | `delete` | `delete`, `retain`, `archive` |
| `dsArchivePrefix` | No | Prefix used when archiving datasets. | `zz` |  |
| `csi.storage.k8s.io/provisioner-secret-name` | Yes | Name of the secret for provisioning. | None | `tns-api-key` |
| `csi.storage.k8s.io/provisioner-secret-namespace` | Yes | Namespace of the provisioning secret. | None | `tns-csi` |
| `csi.storage.k8s.io/controller-expand-secret-name` | Yes | Name of the secret for volume expansion. | None | `tns-api-key` |
| `csi.storage.k8s.io/controller-expand-secret-namespace` | Yes | Namespace of the expansion secret. | None | `tns-csi` |
| `mountPermissions` | No | Permissions applied to mounted volumes. | None | `777` |
| `dsPermissionsMode` | No | Mode for dataset permissions (e.g., `0770`). | None | `0770` |
| `dsPermissionsUser` | No | User ID for dataset ownership. | None | `0` |
| `dsPermissionsGroup` | No | Group ID for dataset ownership. | None | `6000` |
| `shareMaprootUser` | No | User mapped to root for NFS share. | None | `root` |
| `shareMaprootGroup` | No | Group mapped to root for NFS share. | None | `wheel` |
| `shareMapallUser` | No | User mapped for all NFS share accesses. | None |  |
| `shareMapallGroup` | No | Group mapped for all NFS share accesses. | None |  |
| `shareAllowedHosts` | No | Comma-separated list of allowed hostnames for NFS share. | None | `192.168.5.0/24, 192.168.6.0/24` |
| `shareAllowedNetworks` | No | Comma-separated list of allowed networks for NFS share. | None | `192.168.5.0/24, 192.168.6.0/24` |

### Tips
#### `dsNameTemplate` parameter supports the following pv/pvc metadata conversion:
> if `dsNameTemplate` value contains following strings, it would be converted into corresponding pv/pvc name or namespace
 - `${pvc.metadata.name}`
 - `${pvc.metadata.namespace}`
 - `${pv.metadata.name}`
> if `dsNameTemplate` does not contains`${pv.metadata.name}`the driver will append it to the template for uniqueness.
> The maximum length of the dataset name is 200 bytes minus the length of the`dsArchivePrefix` value. If the length exceeds 200, the name is shorten to 200 with the last 15 chars replaced why a unique hash value
#### About the name of the parameters
> Attributes starting with`"ds"`relates to the TrueNAS dataset parameters
> Attributes starting with`"share"`relates to the TrueNAS "Share" settings
> Attributes starting with`"mount"`relates to the file attributes whene the NFS share is mounted in the pod

## Example StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: truenas-csi
provisioner: org.truenas.csi
volumeBindingMode: Immediate
reclaimPolicy: Delete
allowVolumeExpansion: true
mountOptions:
  - hard
  - nfsvers=4.2
  - rsize=1048576
  - wsize=1048576
  - noacl
  - noatime
  - nocto
  - nodiratime
parameters:
  tnsWsUrl: "wss://truenas.server.ip/websocket"
  rootDataset: "POOL-ABCD/CSI"
  dsArchivePrefix: "ab"
  onDelete: "delete"
  csi.storage.k8s.io/provisioner-secret-name: "tns-api-key"
  csi.storage.k8s.io/provisioner-secret-namespace: "tns-csi"
  csi.storage.k8s.io/controller-expand-secret-name: "tns-api-key"
  csi.storage.k8s.io/controller-expand-secret-namespace: "tns-csi"
  mountPermissions: "777"
  dsPermissionsMode: "0770"
  dsPermissionsUser: "0"
  dsPermissionsGroup: "6000"
  shareMaprootUser: "root"
  shareMaprootGroup: "smb"
  shareMapallUser: ""
  shareMapallGroup: ""
  shareAllowedHosts: ""
  shareAllowedNetworks: "192.168.5.0/24 , 192.168.6.0/24"
```

### VolumeSnapshotClass Parameters

The following table  describes the specific parameters available for configuring the volume snapshot class:

| Parameter | Mandatory | Description | Default | Example Value |
|-----------|-----------|-------------|---------|---------------|
| `csi.storage.k8s.io/snapshotter-secret-name` | Yes | Name of the secret for snapshotter. | None | `tns-api-key` |
| `csi.storage.k8s.io/snapshotter-secret-namespace` | Yes | Namespace of the snapshotter secret. | None | `tns-csi` |

## Example VolumeSnapshotClass

```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshotClass
metadata:
  name: tns-csi-vsc
driver: tns.csi.titou10.org
deletionPolicy: Delete
parameters:
  csi.storage.k8s.io/snapshotter-secret-name: tns-api-key
  csi.storage.k8s.io/snapshotter-secret-namespace: tns-csi
```
