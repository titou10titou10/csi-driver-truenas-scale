# NFS CSI driver for Kubernetes to Truenas Scale

### TODO
- write tests
- package
- github repo
- github actions
- publish on csi site: https://kubernetes-csi.github.io/docs/drivers.html

### Improvements
- better delete/archive management?
- review log messages
- review error handling:
  - delete when does not exists
  - create when already exist

### Will not be implemented
 - GetCapacity
   -> Link pod region/rack.. topology to storage capacity associated to that region/rack/...
   -> stil Alpha in k8s v1.31
- ListVolumes + GetVolume
   -> check volumes health
   -> stil Alpha in k8s v1.31
- ListSnapshots
   -> check snapshots health

### To build
- podman installed
