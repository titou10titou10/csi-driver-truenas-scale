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
	"fmt"
	"strings"

	tns "github.com/titou10/csi-driver-truenas-scale/pkg/tns"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"k8s.io/klog/v2"
)

// ControllerServer controller server setting
type ControllerServer struct {
	Driver *Driver
	csi.UnimplementedControllerServer
}

// nfsVolume is an internal representation of a volume created by the provisioner.
type nfsVolume struct {
	id            string // Volume handle
	tnsWsUrl      string // URL for Truenas Scale WS services. Matches paramTnsWsUrl
	rootDataset   string // Base root dataset Matches paramRootDataset
	archivePrefix string //Archive prefix
	onDelete      string // on delete strategy
	size          int64  // size of volume
	dsName        string // dataset name witout root dataset
	pvName        string // pv name given by k8s
}

// nfsSnapshot is an internal representation of a volume snapshot created by the provisioner.
type nfsSnapshot struct {
	id             string // Snapshot handle.
	tnsWsUrl       string // URL for Truenas Scale WS services. Matches paramTnsWsUrl
	rootDataset    string // Base root dataset. Matches paramRootDataset
	sourceDsName   string // Source dataset name
	sourceVolumeId string // Source volume handle (id)
	snapshotName   string // Ssnapshot name
}

// Ordering of elements in the CSI volume id.
// Adding a new element should always go at the end
// before totalIDElements
const (
	idTnsWsUrl = iota
	idRootDataset
	idDsName
	idPvName
	idArchivePrefix
	idOnDelete
	totalIDElements // Always last
)

// Ordering of elements in the CSI snapshot id.
// Adding a new element should always go at the end
// before totalSnapIDElements
const (
	idSnapTnsWsUrl = iota
	idSnapRootDataset
	idSnapName
	idSnapSourceDsName
	totalIDSnapElements // Always last
)

// CreateVolume create a volume
func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	pvName := req.GetName()
	if len(pvName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume name must be provided")
	}

	if err := isValidVolumeCapabilities(req.GetVolumeCapabilities()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var tnsWsUrl = ""
	var rootDataset = ""
	var archivePrefix = DefaultDSArchivePrefix
	var onDelete = cs.Driver.defaultOnDeletePolicy
	var dsNameTemplate = DefaultDsNameTemplate

	reqCapacity := req.GetCapacityRange().GetRequiredBytes()
	parameters := req.GetParameters()

	klog.V(4).Infof("Parameters: %v", parameters)

	if parameters == nil {
		parameters = make(map[string]string)
	}
	// validate parameters (case-insensitive)
	for k, v := range parameters {
		switch strings.ToLower(k) {
		case paramTnsWsUrl:
			tnsWsUrl = v
		case paramRootDataset:
			rootDataset = v
		case paramOnDelete:
			onDelete = v
		case paramDsNameTemplate:
			dsNameTemplate = v
		case pvcNamespaceKey:
		case pvcNameKey:
		case pvNameKey:

		case paramDSPermissionsMode:
		case paramDSPermissionsUser:
		case paramDSPermissionsGroup:
		case paramShareMaprootUser:
		case paramShareMaprootGroup:
		case paramShareMapallUser:
		case paramShareMapallGroup:
		case paramShareAllowedHosts:
		case paramShareAllowedNetworks:
			// no op
		case mountPermissionsField:
			// only used in node mount

		case paramDsArchivePrefix:
			archivePrefix = v

		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid parameter %q in storage class", k)
		}
	}

	if tnsWsUrl == "" {
		return nil, tns.NewCsiError(codes.InvalidArgument, fmt.Errorf("%s is a required parameter", paramTnsWsUrl))
	}
	if rootDataset == "" {
		return nil, tns.NewCsiError(codes.InvalidArgument, fmt.Errorf("%s is a required parameter", paramRootDataset))
	}

	if !isArchivePrefixValid(archivePrefix) {
		return nil, status.Errorf(codes.FailedPrecondition, "Archive prefix can only contain alpha chars")
	}

	if err := validateOnDeleteValue(onDelete); err != nil {
		return nil, err
	}

	apiKey, exists := req.GetSecrets()[apiKeySecretNameKey]
	if !exists || apiKey == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "Secret with 'apiKey' key not found")
	}

	if acquired := cs.Driver.volumeLocks.TryAcquire(pvName); !acquired {
		return nil, status.Errorf(codes.Aborted, volumeOperationAlreadyExistsFmt, pvName)
	}
	defer cs.Driver.volumeLocks.Release(pvName)

	requestedDsname := buildRequestedDsName(tnsWsUrl, rootDataset, archivePrefix, dsNameTemplate, parameters)

	dsName, nfsSharePath, err := tns.CsiVolumeCreate(tnsWsUrl, apiKey, cs.Driver.name, requestedDsname, reqCapacity, parameters)
	if err != nil {
		klog.Errorf("CsiVolumeCreate error: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	nfsVol, err := newNFSVolume(tnsWsUrl, rootDataset, onDelete, archivePrefix, pvName, *dsName, reqCapacity)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.GetVolumeContentSource() != nil {
		vs := req.VolumeContentSource
		switch vs.Type.(type) {
		case *csi.VolumeContentSource_Snapshot:
			csiErr := cs.copyFromSnapshot(req, nfsVol, apiKey)
			if csiErr != nil {
				// TODO cleanup created DS
				return nil, status.Error(codes.Internal, csiErr.Error())
			}
		case *csi.VolumeContentSource_Volume:
			csiErr := cs.copyFromVolume(req, nfsVol, apiKey)
			if csiErr != nil {
				// TODO cleanup created DS
				return nil, status.Error(codes.Internal, csiErr.Error())
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "%v not a proper volume source", vs)
		}
	}

	// Set parameters on PV
	parameters[paramTnsWsUrl] = nfsVol.tnsWsUrl
	parameters[paramNfsSharePath] = *nfsSharePath // Share path use by NodeServer to mount into pods
	parameters[paramDsName] = *dsName

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      nfsVol.id,
			CapacityBytes: 0, // by setting it to zero, Provisioner will use PVC requested size as PV size
			VolumeContext: parameters,
			ContentSource: req.GetVolumeContentSource(),
		},
	}, nil
}

// DeleteVolume delete a volume
func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume id is empty")
	}
	nfsVol, err := getNfsVolFromID(volumeID)
	if err != nil {
		// An invalid ID should be treated as doesn't exist
		klog.Warningf("failed to get nfs volume for volume id %v deletion: %v", volumeID, err)
		return &csi.DeleteVolumeResponse{}, nil
	}

	if nfsVol.onDelete == "" {
		nfsVol.onDelete = cs.Driver.defaultOnDeletePolicy
	}

	apiKey, exists := req.GetSecrets()[apiKeySecretNameKey]
	if !exists || apiKey == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "Secret with 'apiKey' key not found")
	}

	if acquired := cs.Driver.volumeLocks.TryAcquire(volumeID); !acquired {
		return nil, status.Errorf(codes.Aborted, volumeOperationAlreadyExistsFmt, volumeID)
	}
	defer cs.Driver.volumeLocks.Release(volumeID)

	if strings.EqualFold(nfsVol.onDelete, retain) {
		klog.V(2).Infof("DeleteVolume: volume(%s) onDelete is set to retain, Doing nothing", volumeID)
	} else if strings.EqualFold(nfsVol.onDelete, archive) {
		if csiErr := tns.CsiVolumeArchive(nfsVol.tnsWsUrl, apiKey, nfsVol.rootDataset, nfsVol.dsName, nfsVol.archivePrefix); csiErr != nil {
			klog.Errorf("Failed to archive truenas dataset: %v", err)
			return nil, status.Error(csiErr.Code, csiErr.Err.Error())
		}
	} else {
		if csiErr := tns.CsiVolumeDelete(nfsVol.tnsWsUrl, apiKey, nfsVol.dsName); csiErr != nil {
			klog.Errorf("Failed to delete truenas dataset+share+snapshots: %s", csiErr)
			return nil, status.Error(csiErr.Code, csiErr.Err.Error())
		}
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	if len(req.GetName()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateSnapshot name must be provided")
	}
	if len(req.GetSourceVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateSnapshot source volume ID must be provided")
	}
	apiKey, exists := req.GetSecrets()[apiKeySecretNameKey]
	if !exists || apiKey == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "Secret with 'apiKey' key not found")
	}

	srcVol, err := getNfsVolFromID(req.GetSourceVolumeId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to create source volume: %v", err)

	}

	vscParams := req.GetParameters()
	if len(vscParams) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume Snapshot class does not allow extra parameters: %s", vscParams)
	}

	snapName, restoreSize, csiErr := tns.CsiSnapshotCreate(srcVol.tnsWsUrl, apiKey, srcVol.rootDataset, srcVol.dsName, req.GetName())
	if csiErr != nil {
		klog.Errorf("CsiSnapshotCreate error: %s", csiErr)
		return nil, status.Error(csiErr.Code, csiErr.Err.Error())
	}
	snapshot, err := newNFSSnapshot(req.GetName(), *snapName, srcVol, vscParams)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to create nfsSnapshot: %v", err)
	}

	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			SnapshotId:     snapshot.id,
			SourceVolumeId: srcVol.id,
			SizeBytes:      *restoreSize,
			CreationTime:   timestamppb.Now(),
			ReadyToUse:     true,
		},
	}, nil
}

func (cs *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	if len(req.GetSnapshotId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot ID is required for deletion")
	}
	apiKey, exists := req.GetSecrets()[apiKeySecretNameKey]
	if !exists || apiKey == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "Secret with 'apiKey' key not found")
	}

	snapshot, err := getNfsSnapFromID(req.GetSnapshotId())
	if err != nil {
		// An invalid ID should be treated as doesn't exist
		klog.Warningf("failed to get nfs snapshot for id %v deletion: %v", req.GetSnapshotId(), err)
		return &csi.DeleteSnapshotResponse{}, nil
	}

	csiErr := tns.CsiSnapshotDelete(snapshot.tnsWsUrl, apiKey, snapshot.snapshotName)
	if csiErr != nil {
		klog.Errorf("CsiSnapshotDelete error: %s", csiErr)
		return nil, status.Error(csiErr.Code, csiErr.Err.Error())
	}

	return &csi.DeleteSnapshotResponse{}, nil
}

func (cs *ControllerServer) ControllerExpandVolume(_ context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if req.GetCapacityRange() == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity Range missing in request")
	}

	apiKey, exists := req.GetSecrets()[apiKeySecretNameKey]
	if !exists || apiKey == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "Secret with 'apiKey' key not found")
	}

	nfsVol, err := getNfsVolFromID(req.GetVolumeId())
	if err != nil {
		// An invalid ID should be treated as doesn't exist
		klog.Warningf("failed to get volume for id %v expansion: %v", req.GetVolumeId(), err)
		return &csi.ControllerExpandVolumeResponse{}, nil
	}

	volSizeBytes := req.GetCapacityRange().GetRequiredBytes()

	size, csiErr := tns.CsiVolumeExpand(nfsVol.tnsWsUrl, apiKey, nfsVol.rootDataset, nfsVol.dsName, volSizeBytes)
	if csiErr != nil {
		klog.Errorf("CsiDatasetExpand error: %s", csiErr)
		return nil, status.Error(csiErr.Code, csiErr.Err.Error())
	}

	klog.V(2).Infof("ControllerExpandVolume(%s) successfully, currentQuota: %d bytes", req.VolumeId, volSizeBytes)
	return &csi.ControllerExpandVolumeResponse{CapacityBytes: *size}, nil
}

func (cs *ControllerServer) GetCapacity(_ context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")

	// TODO
	// // GetCapacityRequest does not have secrets. impossible to call tns
	// github issue: https://github.com/container-storage-interface/spec/issues/581

	// // Capacity
	// // klog.Infof(">>>> GetCapacity >>>>>>>>>>>>>")
	// var tnsWsUrl = ""
	// var rootDataset = ""
	// parameters := req.GetParameters()
	// if parameters == nil {
	// 	parameters = make(map[string]string)
	// }

	// // validate parameters (case-insensitive)
	// for k, v := range parameters {
	// 	switch strings.ToLower(k) {
	// 	case paramTnsWsUrl:
	// 		tnsWsUrl = v
	// 	case paramRootDataset:
	// 		rootDataset = v
	// 	case paramOnDelete:
	// 	case paramDsNameTemplate:
	// 	case pvcNamespaceKey:
	// 	case pvcNameKey:
	// 	case pvNameKey:

	// 	case paramDSPermissionsMode:
	// 	case paramDSPermissionsUser:
	// 	case paramDSPermissionsGroup:
	// 	case paramShareMaprootUser:
	// 	case paramShareMaprootGroup:
	// 	case paramShareMapallUser:
	// 	case paramShareMapallGroup:
	// 	case paramShareAllowedHosts:
	// 	case paramShareAllowedNetworks:
	// 	case mountPermissionsField:
	// 	case paramDsArchivePrefix:

	// 	default:
	// 		return nil, status.Errorf(codes.InvalidArgument, "invalid parameter %q in storage class", k)
	// 	}
	// }

	// if tnsWsUrl == "" {
	// 	return nil, tns.NewCsiError(codes.InvalidArgument, fmt.Errorf("%s is a required parameter", paramTnsWsUrl))
	// }
	// if rootDataset == "" {
	// 	return nil, tns.NewCsiError(codes.InvalidArgument, fmt.Errorf("%s is a required parameter", paramRootDataset))
	// }

	// apiKey, exists := req.GetSecrets()[apiKeySecretNameKey]
	// if !exists || apiKey == "" {
	// 	return nil, status.Errorf(codes.FailedPrecondition, "Secret with 'apiKey' key not found")
	// }

	// availableCapacity, csiErr := tns.CsiGetCapacity(tnsWsUrl, apiKey, rootDataset)
	// if csiErr != nil {
	// 	klog.Errorf("CsiSnapshotCreate error: %s", csiErr)
	// 	return nil, status.Error(csiErr.Code, csiErr.Err.Error())
	// }
	// return &csi.GetCapacityResponse{
	// 	AvailableCapacity: *availableCapacity,
	// 	//MaximumVolumeSize: &wrappers.Int64Value{Value: 100 * 1024 * 1024 * 1024},
	// 	MinimumVolumeSize: &wrappers.Int64Value{Value: MinimumDatasetSize},
	// }, nil
}

func (cs *ControllerServer) copyFromSnapshot(req *csi.CreateVolumeRequest, dstVol *nfsVolume, apiKey string) *tns.CsiError {
	srcSnapshot, err := getNfsSnapFromID(req.VolumeContentSource.GetSnapshot().GetSnapshotId())
	if err != nil {
		return tns.NewCsiError(codes.NotFound, err)
	}

	csiErr := tns.CsiSnapshotClone(srcSnapshot.tnsWsUrl, apiKey, srcSnapshot.rootDataset, srcSnapshot.snapshotName, dstVol.dsName)
	if csiErr != nil {
		return csiErr
	}

	klog.V(2).Infof("CsiSnapshotClone success")
	return nil
}

func (cs *ControllerServer) copyFromVolume(req *csi.CreateVolumeRequest, dstVol *nfsVolume, apiKey string) *tns.CsiError {
	srcVol, err := getNfsVolFromID(req.GetVolumeContentSource().GetVolume().GetVolumeId())
	if err != nil {
		return tns.NewCsiError(codes.NotFound, err)
	}

	csiErr := tns.CsiDatasetClone(srcVol.tnsWsUrl, apiKey, srcVol.rootDataset, srcVol.dsName, dstVol.dsName)
	if csiErr != nil {
		return csiErr
	}

	klog.V(2).Infof("CsiDatasetClone success. copied %s -> %s", srcVol.dsName, dstVol.dsName)
	return nil
}

func (cs *ControllerServer) ValidateVolumeCapabilities(_ context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if err := isValidVolumeCapabilities(req.GetVolumeCapabilities()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		},
		Message: "",
	}, nil
}

// isValidVolumeCapabilities validates the given VolumeCapability array is valid
func isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) error {
	if len(volCaps) == 0 {
		return fmt.Errorf("volume capabilities missing in request")
	}
	for _, c := range volCaps {
		if c.GetBlock() != nil {
			return fmt.Errorf("block volume capability not supported")
		}
	}
	return nil
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *ControllerServer) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.cscap,
	}, nil
}

func (cs *ControllerServer) ControllerPublishVolume(_ context.Context, _ *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerUnpublishVolume(_ context.Context, _ *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerGetVolume(_ context.Context, x *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	klog.Infof("))))))))))))) ControllerGetVolume: %v", x)
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ListVolumes(_ context.Context, x *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.Infof("))))))))))))) ListVolumes: %v", x)
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerModifyVolume(_ context.Context, x *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	klog.Infof("))))))))))))) ControllerModifyVolume: %v", x)
	return nil, status.Error(codes.Unimplemented, "")
}
func (cs *ControllerServer) ListSnapshots(_ context.Context, x *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	klog.Infof("))))))))))))) ListSnapshots: %v", x)
	return nil, status.Error(codes.Unimplemented, "")
}

// newNFSVolume Convert VolumeCreate parameters to an nfsVolume
func newNFSVolume(tnsWsUrl, rootDataset, onDelete, archivePrefix string, pvName string, dsName string, size int64) (*nfsVolume, *tns.CsiError) {

	vol := &nfsVolume{
		tnsWsUrl:      tnsWsUrl,
		rootDataset:   rootDataset,
		archivePrefix: archivePrefix,
		onDelete:      onDelete,
		size:          size,
		dsName:        dsName,
		pvName:        pvName,
	}

	vol.id = getVolumeIDFromNfsVol(vol)
	return vol, nil
}

// Given a nfsVolume, return a CSI volume id
func getVolumeIDFromNfsVol(vol *nfsVolume) string {
	idElements := make([]string, totalIDElements)
	idElements[idTnsWsUrl] = strings.Trim(vol.tnsWsUrl, "/")
	idElements[idRootDataset] = strings.Trim(vol.rootDataset, "/")
	idElements[idDsName] = strings.Trim(vol.dsName, "/")
	idElements[idPvName] = strings.Trim(vol.pvName, "/")
	idElements[idArchivePrefix] = strings.Trim(vol.archivePrefix, "/")
	idElements[idOnDelete] = vol.onDelete
	return strings.Join(idElements, separator)
}

// <tnsWsUrl>#<rootDataset>#<dsName>#<pvName>#<archiveprefix>#<onDelete>
// wss://truenas.server/websocket # POOL-ZFS02/CSI # POOL-ZFS02/CSI/tns-csi-aaa-pvc-73f86722-fcae-46e3-baa7-d9bd78f5984f # pvc-73f86722-fcae-46e3-baa7-d9bd78f5984f # ab # delete
func getNfsVolFromID(id string) (*nfsVolume, error) {
	var tnsWsUrl, rootDataset, dsName, pvName, archivePrefix, onDelete string
	segments := strings.Split(id, separator)
	tnsWsUrl = segments[0]
	rootDataset = segments[1]
	dsName = segments[2]
	pvName = segments[3]
	archivePrefix = segments[4]
	onDelete = segments[5]
	return &nfsVolume{
		id:            id,
		tnsWsUrl:      tnsWsUrl,
		rootDataset:   rootDataset,
		archivePrefix: archivePrefix,
		onDelete:      onDelete,
		dsName:        dsName,
		pvName:        pvName,
	}, nil
}
func newNFSSnapshot(name string, snapshotName string, srcVol *nfsVolume, params map[string]string) (*nfsSnapshot, error) {
	tnsWsUrl := srcVol.tnsWsUrl
	rootDataset := srcVol.rootDataset

	if tnsWsUrl == "" {
		return nil, fmt.Errorf("%v is a required parameter", paramTnsWsUrl)
	}
	if rootDataset == "" {
		return nil, fmt.Errorf("%v is a required parameter", paramRootDataset)
	}
	snapshot := &nfsSnapshot{
		tnsWsUrl:       tnsWsUrl,
		rootDataset:    rootDataset,
		sourceDsName:   srcVol.dsName,
		sourceVolumeId: srcVol.id,
		snapshotName:   snapshotName,
	}

	snapshot.id = getSnapshotIDFromNfsSnapshot(snapshot)
	return snapshot, nil
}

// <tnsWsUrl>#<rootDataset>#<snapshotName>#<sourceDsName>
// wss://truenas.server/websocket # POOL-ZFS02/CSI # POOL-ZFS02/CSI/tns-csi-aaa-pvc-73f86722-fcae-46e3-baa7-d9bd78f5984f@snapshot-8017dd4d-0d87-450e-a4f3-8922f0347725 # POOL-ZFS02/CSI/tns-csi-aaa-pvc-73f86722-fcae-46e3-baa7-d9bd78f5984f
func getSnapshotIDFromNfsSnapshot(snapshot *nfsSnapshot) string {
	idElements := make([]string, totalIDSnapElements)
	idElements[idSnapTnsWsUrl] = strings.Trim(snapshot.tnsWsUrl, "/")
	idElements[idSnapRootDataset] = strings.Trim(snapshot.rootDataset, "/")
	idElements[idSnapName] = strings.Trim(snapshot.snapshotName, "/")
	idElements[idSnapSourceDsName] = strings.Trim(snapshot.sourceDsName, "/")
	return strings.Join(idElements, separator)
}
func getNfsSnapFromID(id string) (*nfsSnapshot, error) {
	var tnsWsUrl, rootDataset, snapshotName, sourceDsName string
	segments := strings.Split(id, separator)
	tnsWsUrl = segments[0]
	rootDataset = segments[1]
	snapshotName = segments[2]
	sourceDsName = segments[3]

	if len(segments) == totalIDSnapElements {
		return &nfsSnapshot{
			id:           id,
			tnsWsUrl:     tnsWsUrl,
			rootDataset:  rootDataset,
			snapshotName: snapshotName,
			sourceDsName: sourceDsName,
		}, nil
	}

	return &nfsSnapshot{}, fmt.Errorf("failed to create nfsSnapshot from snapshot ID")
}

func buildRequestedDsName(tnsWsUrl, rootDataset, archivePrefix, dsNameTemplate string, params map[string]string) string {

	// 1. if paramDsNameTemplate does not contains pvNameKey: add pvNameKey ad the end of template for uniqueness
	// 2. Add rootDataset at the beginning of the template
	// 3. ensure length < (200 - len(archivePrefix) -1) by replacing the latest 15 chars by hash

	dsNameReplaceMap := map[string]string{
		pvcNamespaceMetadata: params[pvcNamespaceKey],
		pvcNameMetadata:      params[pvcNameKey],
		pvNameMetadata:       params[pvNameKey],
	}

	if !strings.Contains(strings.ToLower(dsNameTemplate), strings.ToLower(pvNameMetadata)) {
		dsNameTemplate += "-" + pvNameMetadata
	}
	baseName := replaceWithMap(dsNameTemplate, dsNameReplaceMap)
	requestedDsName := truncateWithHash(rootDataset+"/"+baseName, TruenassDsMaxLength-len(archivePrefix))

	klog.V(3).Infof("buildRequestedDsName root: %s template: %s base: %s requestedDsName: %s", rootDataset, dsNameTemplate, baseName, requestedDsName)

	return requestedDsName
}
