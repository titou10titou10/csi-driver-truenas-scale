# NFS CSI driver for Kubernetes to Truenas Scale

### TODO
- write tests
- package
- publish on csi site: https://kubernetes-csi.github.io/docs/drivers.html
- implements Block storage via iSCSI

### Improvements
- better delete/archive management? -> rename dataset currently not implemented via wss..
- review log messages
- review error handling:
  - delete when does not exists
  - create when already exist

### Will not be implemented
- Ephemeral Local Volumes
  - "A CSI driver is not suitable for CSI ephemeral inline volumes when..provisioning is not local to the node"

- Storage Capacity Tracking
  - Link pod region/rack.. topology to storage capacity associated to that region/rack/...
  - Still Alpha in k8s v1.31

- Volume Health Monitoring Feature
  - Still Alpha in k8s v1.31

- ListVolumes + GetVolume
  - check volumes health
  - Still Alpha in k8s v1.31

- ListSnapshots
  - check snapshots health

- Volume Limits
  - not relevant

- Token Requests
  - not relevant
