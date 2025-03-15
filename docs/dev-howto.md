# CSI driver for Kubernetes for TrueNAS Scale development guide

### Requirements

- go 1.24.1+ installed
- podman is installed

## Build
 - Clone repo
```console
$ git clone https://github.com/titou10titou10/csi-driver-truenas-scale 
$ cd csi-driver-truenas-scale
```

 - Build the CSI driver
```console
$ cd csi-driver-truenas-scale
$ make build
```

 - Package the Helm Chart
```console
$ cd csi-driver-truenas-scale
$ make helm-package
```

## Push artefefacts

 - Push the driver image
```console
$ cd csi-driver-truenas-scale
$ make build-and-push
```

 - Push the helm chart
```console
$ cd csi-driver-truenas-scale
$ make helm-push
```

## Deployment on local kubernetes

- login to kubernetes
- create a new namespace to hold the driver
- create a`"values.yaml"` manifest based on [values.yaml](./charts/dev/tns-csi-driver/values.yanl)
```console
$ cd csi-driver-truenas-scale
$ helm upgrade -i tns.csi.titou10.org -f <your values.yaml> ./charts/dev/tns-csi-driver
```

TBD
TBD
TBD
 
## How to test CSI driver in local environment

Install `csc` tool according to https://github.com/rexray/gocsi/tree/master/csc
```console
$ mkdir -p $GOPATH/src/github.com
$ cd $GOPATH/src/github.com
$ git clone https://github.com/rexray/gocsi.git
$ cd rexray/gocsi/csc
$ make build
```

#### Start CSI driver locally
```console
$ cd csi-driver-truenas-scale
$ ./bin/tnsplugin --endpoint unix:///tmp/csi.sock --nodeid CSINode -v=5 &
```

#### 0. Set environment variables
```console
$ cap="1,mount,"
$ volname="test-$(date +%s)"
$ volsize="2147483648"
$ endpoint="unix:///tmp/csi.sock"
$ target_path="/tmp/targetpath"
$ params="tnsWsUrl=wss://127.0.0.1/websocket,rootDataset=/"
```

#### 1. Get plugin info
```console
$ csc identity plugin-info --endpoint "$endpoint"
"tns.csi.titou10.org"    "v0.9.0"
```

#### 2. Create a new volume
```console
$ value="$(csc controller new --endpoint "$endpoint" --cap "$cap" "$volname" --req-bytes "$volsize" --params "$params")"
$ sleep 15
$ volumeid="$(echo "$value" | awk '{print $1}' | sed 's/"//g')"
$ echo "Got volume id: $volumeid"
```

#### 3. Publish a volume
```
$ csc node publish --endpoint "$endpoint" --cap "$cap" --vol-context "$params" --target-path "$target_path" "$volumeid"
```

#### 4. Unpublish a volume
```console
$ csc node unpublish --endpoint "$endpoint" --target-path "$target_path" "$volumeid"
```

#### 6. Validate volume capabilities
```console
$ csc controller validate-volume-capabilities --endpoint "$endpoint" --cap "$cap" "$volumeid"
```

#### 7. Delete the volume
```console
$ csc controller del --endpoint "$endpoint" "$volumeid" --timeout 10m
```

#### 8. Get NodeID
```console
$ csc node get-info --endpoint "$endpoint"
CSINode
```

