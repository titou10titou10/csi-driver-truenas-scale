# Install the TrueNAS Scale CSI driver with Helm

## Overview

This Helm chart deploys the Truenas Scale CSI driver, enabling dynamic provisioning of NFS volumes with ZFS snapshots in

## Prerequisites
 - [install Helm](https://helm.sh/docs/intro/quickstart/#install-helm)
 - the CSI snapshot controller must be installed on the cluster, Modern version install it by default, 
   if not, visit the [external-snapshotter](https://github.com/kubernetes-csi/external-snapshotter) github site or visit your k8s implementation

## Install
Either create a`"values.yml"`file or set chart parameters with`"--set..."`parameters
```console
helm upgrade -i tns.csi.titou10.org -n <namespace> -f <your values.yaml> oci://ghcr.io/titou10titou10/tns-csi-driver:v0.9.0
```

## Uninstall
```console
helm uninstall tns.csi.titou10.org -n <namespace>
```

## Tips
 - run controller on control plane node: `--set controller.runOnControlPlane=true`
 - set replica of controller as `2`: `--set controller.replicas=2`
 - Microk8s based kubernetes recommended settings(refer to https://microk8s.io/docs/nfs):
    - `--set controller.dnsPolicy=ClusterFirstWithHostNet`
    - `--set kubeletDir="/var/snap/microk8s/common/var/lib/kubelet"` : sets correct path to microk8s kubelet even
      though a user has a folder link to it.
 - on OpenShift/OKD, the controllers need privileged access to mount the nfs shares. R»un the following
    - `oc adm policy add-scc-to-user privileged -n <namespace> -z <CSI controller service account>`
    - `oc adm policy add-scc-to-user privileged -n <namespace> -z <node controller service account>`


## Chart Parameters

The following table lists the configurable parameters of the chart and their default values:

| Parameter                          | Description                             | Default                                       |
| ---------------------------------- | --------------------------------------- | --------------------------------------------- |
| `image.tnsplugin.repository`       | Repository for the CSI plugin image     | `ghcr.io/titou10titou10/tnsplugin`            |
| `image.tnsplugin.tag`              | Tag for the CSI plugin image            | `dev`                                      |
| `image.tnsplugin.pullPolicy`       | Image pull policy                       | `IfNotPresent`                                |
| `image.csiProvisioner.repository`  | Repository for CSI provisioner          | `registry.k8s.io/sig-storage/csi-provisioner` |
| `image.csiProvisioner.tag`         | Tag for CSI provisioner                 | `v5.2.0`                                      |
| `image.csiProvisioner.pullPolicy`  | Image pull policy                       | `IfNotPresent`                                |
| `image.csiResizer.repository`      | Repository for CSI resizer              | `registry.k8s.io/sig-storage/csi-resizer`     |
| `image.csiResizer.tag`             | Tag for CSI resizer                     | `v1.13.1`                                     |
| `image.csiResizer.pullPolicy`      | Image pull policy                       | `IfNotPresent`                                |
| `image.csiSnapshotter.repository`  | Repository for CSI snapshotter          | `registry.k8s.io/sig-storage/csi-snapshotter` |
| `image.csiSnapshotter.tag`         | Tag for CSI snapshotter                 | `v8.2.0`                                      |
| `image.csiSnapshotter.pullPolicy`  | Image pull policy                       | `IfNotPresent`                                |
| `serviceAccount.create`            | Create service accounts                 | `true`                                        |
| `serviceAccount.controller`        | Name of controller service account      | `tns-csi-controller-sa`                       |
| `serviceAccount.node`              | Name of node service account            | `tns-csi-node-sa`                             |
| `rbac.create`                      | Create RBAC roles                       | `true`                                        |
| `rbac.namePrefix`                  | Prefix for RBAC roles                   | `tns-csi`                                     |
| `driver.name`                      | Name of the CSI driver                  | `tns.csi.titou10.org`                         |
| `driver.mountPermissions`          | Mount permissions                       | `0`                                           |
| `feature.enableFSGroupPolicy`      | Enable FSGroup policy                   | `true`                                        |
| `kubeletDir`                       | Path to kubelet directory               | `/var/lib/kubelet`                            |
| `customLabels`                     | Custom labels                           | `{}`                                          |
| `controller.replicas`              | Number of controller replicas           | `1`                                           |
| `controller.strategyType`          | Deployment strategy type                | `Recreate`                                    |
| `controller.runOnMaster`           | Run on master nodes                     | `false`                                       |
| `controller.runOnControlPlane`     | Run on control-plane nodes              | `false`                                       |
| `controller.enableSnapshotter`     | Enable snapshotter                      | `true`                                        |
| `controller.logLevel`              | Log level for controller                | `5`                                           |
| `controller.workingMountDir`       | Working mount directory                 | `/tmp`                                        |
| `controller.dnsPolicy`             | DNS policy for controller               | `ClusterFirstWithHostNet`                     |
| `controller.defaultOnDeletePolicy` | Default volume deletion policy          | `delete`                                      |
| `node.logLevel`                    | Log level for node                      | `5`                                           |
| `node.dnsPolicy`                   | DNS policy for node                     | `ClusterFirstWithHostNet`                     |
| `node.maxUnavailable`              | Maximum unavailable nodes during update | `1`                                           |
| `imagePullSecrets`                 | Image pull secrets                      | `[]`                                          |
| `tnsApiKeySecret.create`           | Create TrueNAS API key secret           | `false`                                       |
| `tnsApiKeySecret.name`             | Name of TrueNAS API key secret          | `truenas-apikey`                              |
| `storageClass.create`              | Create a storage class                  | `false`                                       |
| `storageClass.name`                | Storage class name                      | `tns-csi-sc`                                  |
| `volumeSnapshotClass.create`       | Create a volume snapshot class          | `false`                                       |
| `volumeSnapshotClass.name`         | Volume snapshot class name              | `tns-csi-vsc`                                 |

## troubleshooting
 - Add `--wait -v=5 --debug` in `helm install` command to get detailed error
 - Use `kubectl describe` to acquire more info
