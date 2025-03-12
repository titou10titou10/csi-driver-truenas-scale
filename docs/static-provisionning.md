## Static Provisionning
The csi driver support static provisionning.

Prerequisites:
- create a dataset in TrueNAS Scale
- create a share for the dataset
- create a PersistentVolume. 

In addition to the standard`PersistentVolume` parameters, the following attributes are required:
  - `driver`: <name of the driver, eg `tns.csi.titou10.org`>
  - `volumeAttributes.nfssharepath`: the name of the share in TrueNAS
  - `volumeHandle`: a string composed like this:
  ```console
     {truenas-ws-url}#{rootDataset}#{full datasetName}#{archivePrefix}#{ondelete}
     eg: 'wss://truenas.server/websocket#POOL-ZFS02/CSI#POOL-ZFS02/CSI/abcdef#ab#delete'
  ```

Example:  

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: test-pv
spec:
  accessModes:
    - ReadWriteMany
  mountOptions:
  - hard
  - nfsvers=4.2
  - rsize=1048576
  - wsize=1048576
  capacity:
    storage: 1Gi
  csi:
    # volumeHandle format: {truenas-ws-url}#{rootDataset}#{full datasetName}#{archivePrefix}#{ondelete}
    # make sure this value is unique in the cluster
    driver: tns.csi.titou10.org
    volumeHandle: 'wss://truenas.server/websocket#POOL-ZFS02/CSI#POOL-ZFS02/CSI/abcdef#ab#delete'
    volumeAttributes:
      nfssharepath: /mnt/POOL-ZFS02/CSI/abcdef
```