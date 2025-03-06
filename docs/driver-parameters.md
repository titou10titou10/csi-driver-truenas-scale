## Truenas Configuration
> This driver requires a configured Truenas Scale server with the following:
- a "root" dataset in truenas scale that will hold children dataset for PVs 
- the NFS sharing service must be enabled
- a user with the right to create datasets inside the "root" dataset, take snapshots and create nfs shares
- an api token for the user

## Driver Parameters

### storage class usage (dynamic provisioning)
> [`StorageClass` example](../deploy/example/storageclass-nfs.yaml)

Name | Meaning | Example Value | Mandatory | Default value
--- | --- | --- | --- | ---
server | NFS Server address | domain name `nfs-server.default.svc.cluster.local` <br>or IP address `127.0.0.1` | Yes |
share | NFS share path | `/` | Yes |
subDir | sub directory under nfs share |  | No | if sub directory does not exist, this driver would create a new one
mountPermissions | mounted folder permissions. The default is `0`, if set as non-zero, driver will perform `chmod` after mount |  | No |
onDelete | when volume is deleted, keep the directory if it's `retain` | `delete`(default), `retain`, `archive`  | No | `delete`

 - VolumeID(`volumeHandle`) is the identifier of the volume handled by the driver, format of VolumeID:
```
{nfs-server-address}#{sub-dir-name}#{share-name}
```
> example: `nfs-server.default.svc.cluster.local/share#subdir#`

### PV/PVC usage (static provisioning)
> [`PersistentVolume` example](../deploy/example/pv-nfs-csi.yaml)

Name | Meaning | Example Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeHandle | Specify a value the driver can use to uniquely identify the share in the cluster. | A recommended way to produce a unique value is to combine the nfs-server address, sub directory name and share name: `{nfs-server-address}#{sub-dir-name}#{share-name}`. | Yes |
volumeAttributes.server | NFS Server address | domain name `nfs-server.default.svc.cluster.local` <br>or IP address `127.0.0.1` | Yes |
volumeAttributes.share | NFS share path | `/` |  Yes  |
volumeAttributes.mountPermissions | mounted folder permissions. The default is `0`, if set as non-zero, driver will perform `chmod` after mount |  | No |

### Tips
#### `subDir` parameter supports following pv/pvc metadata conversion
> if `subDir` value contains following strings, it would be converted into corresponding pv/pvc name or namespace
 - `${pvc.metadata.name}`
 - `${pvc.metadata.namespace}`
 - `${pv.metadata.name}`

