// Copyright (C) 2025 Denis Forveille titou10.titou10@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"os"

	"github.com/titou10/csi-driver-truenas-scale/pkg/csi"
	"k8s.io/klog/v2"
)

var (
	endpoint              = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeID                = flag.String("nodeid", "", "node id")
	mountPermissions      = flag.Uint64("mount-permissions", 0, "mounted folder permissions")
	driverName            = flag.String("drivername", csi.DefaultDriverName, "name of the driver")
	defaultOnDeletePolicy = flag.String("default-ondelete-policy", "", "default policy for deleting datasets when deleting a volume")
)

func main() {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "true")
	flag.Parse()
	if *nodeID == "" {
		klog.Warning("nodeid is empty")
	}

	handle()
	os.Exit(0)
}

func handle() {
	driverOptions := csi.DriverOptions{
		NodeID:                *nodeID,
		DriverName:            *driverName,
		Endpoint:              *endpoint,
		MountPermissions:      *mountPermissions,
		DefaultOnDeletePolicy: *defaultOnDeletePolicy,
	}
	d := csi.NewDriver(&driverOptions)
	d.Run(false)
}
