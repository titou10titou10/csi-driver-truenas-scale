## Driver Parameters
The csi driver does not required any specific parameters

## Example CSIDriver

```yaml
kind: CSIDriver
apiVersion: storage.k8s.io/v1
metadata:
  name: tns.csi.titou10.org
spec:
  attachRequired: false
  volumeLifecycleModes:
    - Persistent
  storageCapacity: false # default
  fsGroupPolicy: File
  requiresRepublish: false # default
  seLinuxMount: false # default

```