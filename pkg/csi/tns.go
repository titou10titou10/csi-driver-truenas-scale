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

package csi

import (
	"runtime"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	tns "github.com/titou10/csi-driver-truenas-scale/pkg/tns"
	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"
)

// DriverOptions defines driver parameters specified in driver deployment
type DriverOptions struct {
	NodeID                string
	DriverName            string
	Endpoint              string
	MountPermissions      uint64
	DefaultOnDeletePolicy string
}

type Driver struct {
	name                  string
	nodeID                string
	version               string
	endpoint              string
	mountPermissions      uint64
	defaultOnDeletePolicy string

	//ids *identityServer
	ns          *NodeServer
	cscap       []*csi.ControllerServiceCapability
	nscap       []*csi.NodeServiceCapability
	volumeLocks *VolumeLocks
}

const (
	DefaultDriverName      = "tns.csi.titou10.org"
	DefaultDsNameTemplate  = "${pvc.metadata.namespace}-${pvc.metadata.name}-${pv.metadata.name}"
	DefaultDSArchivePrefix = "zz"
	TruenassDsMaxLength    = 200
	MinimumDatasetSize     = 1 * 1024 * 1024 * 1024 // 1 GB

	// Secret key for Truenas Scale api key
	apiKeySecretNameKey = "apiKey"

	// Params set on PV
	paramDsName       = "dsname"
	paramNfsSharePath = "nfssharepath"

	// Storage class parameters
	paramTnsWsUrl        = "tnswsurl"
	paramRootDataset     = "rootdataset"
	paramOnDelete        = "ondelete"
	paramDsNameTemplate  = "dsnametemplate"
	paramDsArchivePrefix = "dsarchiveprefix"

	// linux mount directory permission
	mountPermissionsField = "mountpermissions"

	// truenas dataset properties
	paramDSPermissionsMode  = "dspermissionsmode"
	paramDSPermissionsUser  = "dspermissionsuser"
	paramDSPermissionsGroup = "dspermissionsgroup"

	// truneas share properties
	paramShareMaprootUser     = "sharemaprootuser"
	paramShareMaprootGroup    = "sharemaprootgroup"
	paramShareMapallUser      = "sharemapalluser"
	paramShareMapallGroup     = "sharemapallgroup"
	paramShareAllowedHosts    = "shareallowedhosts"
	paramShareAllowedNetworks = "shareallowednetworks"

	pvcNameKey           = "csi.storage.k8s.io/pvc/name"
	pvcNamespaceKey      = "csi.storage.k8s.io/pvc/namespace"
	pvNameKey            = "csi.storage.k8s.io/pv/name"
	pvcNameMetadata      = "${pvc.metadata.name}"
	pvcNamespaceMetadata = "${pvc.metadata.namespace}"
	pvNameMetadata       = "${pv.metadata.name}"
)

func NewDriver(options *DriverOptions) *Driver {
	klog.V(2).Infof("Driver: %v version: %v", options.DriverName, driverVersion)

	n := &Driver{
		name:                  options.DriverName,
		version:               driverVersion,
		nodeID:                options.NodeID,
		endpoint:              options.Endpoint,
		mountPermissions:      options.MountPermissions,
		defaultOnDeletePolicy: options.DefaultOnDeletePolicy,
	}

	n.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
		csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,

		//csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		//csi.ControllerServiceCapability_RPC_GET_VOLUME,
		//csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		// Capacity
		// csi.ControllerServiceCapability_RPC_GET_CAPACITY,
	})

	n.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		csi.NodeServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
		csi.NodeServiceCapability_RPC_UNKNOWN,
	})
	n.volumeLocks = NewVolumeLocks()

	return n
}

func NewNodeServer(n *Driver, mounter mount.Interface) *NodeServer {
	return &NodeServer{
		Driver:  n,
		mounter: mounter,
	}
}

func (n *Driver) Run(testMode bool) {
	versionMeta, err := GetVersionYAML(n.name)
	if err != nil {
		klog.Fatalf("%v", err)
	}
	klog.V(2).Infof("\nDRIVER INFORMATION:\n-------------------\n%s\n\nStreaming logs below:", versionMeta)

	mounter := mount.New("")
	if runtime.GOOS == "linux" {
		// MounterForceUnmounter is only implemented on Linux now
		mounter = mounter.(mount.MounterForceUnmounter)
	}
	n.ns = NewNodeServer(n, mounter)
	s := NewNonBlockingGRPCServer()

	s.Start(n.endpoint,
		NewDefaultIdentityServer(n),
		// NFS plugin has not implemented ControllerServer
		// using default controllerserver.
		NewControllerServer(n),
		n.ns,
		testMode)

	// Start background wss connection cleaning
	tns.TNSStartWSSCleanupRoutine(10*time.Minute, 10*time.Minute)

	s.Wait()
}

func (n *Driver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability
	for _, c := range cl {
		csc = append(csc, NewControllerServiceCapability(c))
	}
	n.cscap = csc
}

func (n *Driver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) {
	var nsc []*csi.NodeServiceCapability
	for _, n := range nl {
		nsc = append(nsc, NewNodeServiceCapability(n))
	}
	n.nscap = nsc
}

func IsCorruptedDir(dir string) bool {
	_, pathErr := mount.PathExists(dir)
	return pathErr != nil && mount.IsCorruptedMnt(pathErr)
}

// replaceWithMap replace key with value for str
func replaceWithMap(str string, m map[string]string) string {
	for k, v := range m {
		if k != "" {
			str = strings.ReplaceAll(str, k, v)
		}
	}
	return str
}
