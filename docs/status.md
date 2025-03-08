# NFS CSI driver for Kubernetes to Truenas Scale

### TODO
- write tests
- package
- publish on csi site: https://kubernetes-csi.github.io/docs/drivers.html

### Improvements
- better delete/archive management?
- review log messages
- review error handling:
  - delete when does not exists
  - create when already exist

### Will not be implemented
 - GetCapacity
   - Link pod region/rack.. topology to storage capacity associated to that region/rack/...
   - still Alpha in k8s v1.31
- ListVolumes + GetVolume
   - check volumes health
   - still Alpha in k8s v1.31
- ListSnapshots
   -> check snapshots health
