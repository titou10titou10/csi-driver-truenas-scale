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

package tns

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

func CsiVolumeCreate(tnsWsUrl string, apiKey string, driverName string, dsName string, reqCapacity int64, parameters map[string]string) (*string, *string, *CsiError) {
	klog.V(2).Infof("*** CsiVolumeCreate tnsWsUrl: %s dsName: %s reqCapacity: %d", tnsWsUrl, dsName, reqCapacity)
	defer klog.V(2).Info("*** CsiVolumeCreate")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return nil, nil, csiErr
	}
	defer ReleaseClient(client)

	ds, csiErr := TNSDatasetCreate(client, driverName, dsName, reqCapacity, parameters)
	if csiErr != nil {
		return nil, nil, logAndReturnError("Failed to create dataset", csiErr)
	}

	if csiErr := TNSDatasetSetPermissions(client, ds.MountPoint, parameters); csiErr != nil {
		cleanupDataset(client, ds.Name)
		return nil, nil, logAndReturnError("Failed to set permissions", csiErr)
	}

	nfsSharePath, csiErr := TNSShareNfsCreate(client, ds.MountPoint, parameters)
	if csiErr != nil {
		cleanupDataset(client, ds.Name)
		return nil, nil, logAndReturnError("Failed to create NFS share", csiErr)
	}

	klog.Info("++ Dataset and NFS share created successfully")
	return &dsName, nfsSharePath, nil
}

func CsiVolumeDelete(tnsWsUrl string, apiKey string, dsName string) *CsiError {
	klog.V(2).Infof("*** CsiVolumeDelete tnsWsUrl: %s dsName: %s", tnsWsUrl, dsName)
	defer klog.V(2).Info("*** CsiVolumeDelete")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return csiErr
	}
	defer ReleaseClient(client)

	// delete ds + share + snapshots
	csiErr = TNSDatasetDelete(client, dsName)
	if csiErr != nil {
		klog.Errorf("Volume delete failed:: %s", csiErr)
		return csiErr
	}

	klog.V(2).Info("++ Dataset delete successful")
	return nil
}

func CsiVolumeArchive(tnsWsUrl string, apiKey string, rootDataset string, dsName string, archivePrefix string) *CsiError {
	klog.V(2).Infof("*** CsiVolumeArchive tnsWsUrl: %s rootDataset: %s dsName: %s archivePrefix: %s", tnsWsUrl, rootDataset, dsName, archivePrefix)
	defer klog.V(2).Info("*** CsiVolumeArchive")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return csiErr
	}
	defer ReleaseClient(client)

	baseDsName := strings.Replace(dsName, rootDataset+"/", "", 1)
	tempSnapshotName := archivePrefix + "_" + baseDsName
	archiveDsName := rootDataset + "/" + archivePrefix + "_" + baseDsName

	// Take snapshot
	snapshot, csiErr := TNSSnapshotCreate(client, dsName, tempSnapshotName)
	if csiErr != nil {
		klog.Errorf("Volume archive failed during snapshot creation: %s", csiErr)
		return csiErr
	}

	// Restore snapshot into new archive ds
	csiErr = TNSSnapshotClone(client, snapshot.Name, archiveDsName)
	if csiErr != nil {
		klog.Errorf("Volume archive failed during snapshot cloning: %s", csiErr)
		_, csiErr2 := TNSSnapshotDelete(client, snapshot.Name)
		if csiErr2 != nil {
			klog.Errorf("Snapshot cleanup failed. Ignoring: %s", csiErr2)
		}
		return csiErr
	}

	// Promote archive ds
	csiErr = TNSDatasetPromote(client, archiveDsName)
	if csiErr != nil {
		klog.Errorf("Volume archive failed during dataset promotion: %s", csiErr)
		_, csiErr2 := TNSSnapshotDelete(client, snapshot.Name)
		if csiErr2 != nil {
			klog.Errorf("Snapshot cleanup failed. Ignoring: %s", csiErr2)
		}
		csiErr2 = TNSDatasetDelete(client, archiveDsName)
		if csiErr2 != nil {
			klog.Errorf("Archive Dataset cleanup failed. Ignoring: %s", csiErr2)
		}
		return csiErr
	}

	// Delete base ds + snapshots + share
	csiErr = TNSDatasetDelete(client, dsName)
	if csiErr != nil {
		klog.Errorf("Volume archive failed during dataset deletion: %s", csiErr)
		_, csiErr2 := TNSSnapshotDelete(client, snapshot.Name)
		if csiErr2 != nil {
			klog.Errorf("Snapshot cleanup failed. Ignoring: %s", csiErr2)
		}
		csiErr2 = TNSDatasetDelete(client, archiveDsName)
		if csiErr2 != nil {
			klog.Errorf("Archive Dataset cleanup failed. Ignoring: %s", csiErr2)
		}
		return csiErr
	}

	// Delete Snapshot on archive
	_, csiErr = TNSDatasetDestroySnapshotsJob(client, archiveDsName)
	if csiErr != nil {
		klog.Warningf("Delete Snapshot on archive dataset failed. Continue: %v", csiErr)
	}

	klog.V(2).Info("++ Volume archive completed successfully")
	return nil
}

func CsiDatasetClone(tnsWsUrl string, apiKey string, rootDataset string, srcDsName, destDsName string) *CsiError {
	klog.V(2).Infof("*** CsiDatasetClone tnsWsUrl: %s rootDataset: %s srcDsName: %s destDsName: %s", tnsWsUrl, rootDataset, srcDsName, destDsName)
	defer klog.V(2).Info("*** CsiDatasetClone")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return csiErr
	}
	defer ReleaseClient(client)

	snapshotName := uuid.New().String()
	tempSnapshot, csiErr := TNSSnapshotCreate(client, srcDsName, snapshotName)
	if csiErr != nil {
		return csiErr
	}

	// Start Replication Job
	jobID, csiErr := TNSOneTimeReplicationJob(client, srcDsName, snapshotName, destDsName)
	if csiErr != nil {
		// Try to cleanup the freshly created dataset
		csiErr2 := TNSDatasetDelete(client, destDsName)
		if csiErr2 != nil {
			klog.Warningf("Dataset delete/cleanup failed. Continue: %v", csiErr2)
		}
		return csiErr
	}

	// Wait for Completion
	if csiErr := waitForJobCompletion(client, jobID); csiErr != nil {
		// Try to cleanup the freshly created dataset
		csiErr2 := TNSDatasetDelete(client, destDsName)
		if csiErr2 != nil {
			klog.Warningf("Dataset delete/cleanup failed. Continue: %v", csiErr2)
		}
		return csiErr
	}

	// Delete Source Snapshot
	_, csiErr = TNSSnapshotDelete(client, tempSnapshot.Name)
	if csiErr != nil {
		klog.Warningf("Delete Snapshot created for replication failed. Continue: %v", csiErr)
	}

	// Delete target Snapshot
	_, csiErr = TNSDatasetDestroySnapshotsJob(client, destDsName)
	if csiErr != nil {
		klog.Warningf("Delete Snapshot created for replication failed. Continue: %v", csiErr)
	}

	klog.Info("Dataset clone successfull")
	return nil
}

func CsiGetCapacity(tnsWsUrl string, apiKey string, dsName string) (*int64, *CsiError) {
	klog.V(2).Infof("*** CsiGetCapacity tnsWsUrl: %s dsName: %s", tnsWsUrl, dsName)
	defer klog.V(2).Info("*** CsiGetCapacity")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return nil, csiErr
	}
	defer ReleaseClient(client)

	// delete ds + share + snapshots
	ds, csiErr := TNSDatasetGet(client, dsName)
	if csiErr != nil {
		klog.Errorf("Get Capacity get failed:: %s", csiErr)
		return nil, csiErr
	}

	parsed, ok := ds.Available.Parsed.(float64)
	if !ok {
		csiErr := NewCsiError(codes.Internal, fmt.Errorf("Error parsing Available value. Could not assert that '%v' is float64", ds.Available.Parsed))
		klog.Errorf("Get Capacity get failed:: %s", csiErr)
		return nil, csiErr
	}

	availableCapacity := int64(parsed)

	klog.V(2).Infof("++ Capacity get successful: %d", availableCapacity)
	return &availableCapacity, nil
}

func CsiSnapshotClone(tnsWsUrl string, apiKey string, rootDataset string, srcSnapshotName string, destDsName string) *CsiError {
	klog.V(2).Infof("*** CsiSnapshotClone tnsWsUrl: %s rootDataset: %s srcSnapshotName: %s destDsName: %s", tnsWsUrl, rootDataset, srcSnapshotName, destDsName)
	defer klog.V(2).Info("*** CsiSnapshotClone")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return csiErr
	}
	defer ReleaseClient(client)

	snapshotParts := strings.Split(srcSnapshotName, "@")
	dsName := snapshotParts[0]
	snapshotName := snapshotParts[1]

	// Start Replication Job
	jobID, csiErr := TNSOneTimeReplicationJob(client, dsName, snapshotName, destDsName)
	if csiErr != nil {
		// Try to cleanup the freshly created dataset
		csiErr2 := TNSDatasetDelete(client, destDsName)
		if csiErr2 != nil {
			klog.Warningf("Dataset delete/cleanup failed. Continue: %v", csiErr2)
		}
		return csiErr
	}

	// Wait for Completion
	if csiErr := waitForJobCompletion(client, jobID); csiErr != nil {
		// Try to cleanup the freshly created dataset
		csiErr2 := TNSDatasetDelete(client, destDsName)
		if csiErr2 != nil {
			klog.Warningf("Dataset delete/cleanup failed. Continue: %v", csiErr2)
		}
		return csiErr
	}

	// Delete target snapshots
	_, csiErr = TNSDatasetDestroySnapshotsJob(client, destDsName)
	if csiErr != nil {
		klog.Warningf("Delete Snapshot created for replication failed. Continue: %v", csiErr)
	}

	klog.Info("Snapshot clone successfull")
	return nil
}

func CsiSnapshotCreate(tnsWsUrl string, apiKey string, rootDataset string, dsName string, snapshotName string) (*string, *int64, *CsiError) {
	klog.V(2).Infof("*** CsiSnapshotCreate tnsWsUrl: %s rootDataset: %s dsName: %s snapshotName: %s", tnsWsUrl, rootDataset, dsName, snapshotName)
	defer klog.V(2).Info("*** CsiSnapshotCreate")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return nil, nil, csiErr
	}
	defer ReleaseClient(client)

	snapshot, csiErr := TNSSnapshotCreate(client, dsName, snapshotName)
	if csiErr != nil {
		klog.Errorf("Snapshot creation failed: %s", csiErr)
		return nil, nil, csiErr
	}
	var restoreSize int64 = 0
	if parsed, ok := snapshot.Properties.Referenced.Parsed.(float64); ok {
		restoreSize = int64(parsed)
	}

	klog.V(2).Infof("++ Snapshot created successfully. Restore size: %d", restoreSize)
	return &snapshot.Name, &restoreSize, nil
}

func CsiSnapshotDelete(tnsWsUrl string, apiKey string, snapshotName string) *CsiError {
	klog.V(2).Infof("*** CsiSnapshotDelete tnsWsUrl: %s snapshotName: %s", tnsWsUrl, snapshotName)
	defer klog.V(2).Info("*** CsiSnapshotDelete")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return csiErr
	}
	defer ReleaseClient(client)

	res, csiErr := TNSSnapshotDelete(client, snapshotName)
	if csiErr != nil {
		klog.Errorf("Snapshot delete failed: %s", csiErr)
		return csiErr
	}

	klog.V(2).Infof("++ Snapshot delete successful: %t", *res)
	return nil
}

func CsiVolumeExpand(tnsWsUrl string, apiKey string, rootDataset string, dsName string, newSize int64) (*int64, *CsiError) {
	klog.V(2).Infof("*** CsiVolumeExpand tnsWsUrl: %s rootDataset: %s dsName: %s newSize: %d", tnsWsUrl, rootDataset, dsName, newSize)
	defer klog.V(2).Info("*** CsiVolumeExpand")

	client, csiErr := GetClient(tnsWsUrl, apiKey, true)
	if csiErr != nil {
		return nil, csiErr
	}
	defer ReleaseClient(client)

	res, csiErr := TNSDatasetSetSize(client, dsName, newSize)
	if csiErr != nil {
		klog.Errorf("Dataset expand failed: %s", csiErr)
		return nil, csiErr
	}

	var size int64 = 0
	if parsed, ok := res.RefQuota.Parsed.(float64); ok {
		size = int64(parsed)
	}

	klog.V(2).Infof("++ Dataset expanded successfully. New size: %d", size)
	return &size, nil
}

// -------
// Helpers
// -------

func cleanupDataset(client *Client, dsName string) {
	if err := TNSDatasetDelete(client, dsName); err != nil {
		klog.Warningf("Dataset cleanup failed: %v", err)
	}
}
func logAndReturnError(msg string, err *CsiError) *CsiError {
	klog.Errorf("%s: %s", msg, err)
	return err
}

func waitForJobCompletion(client *Client, jobID *int) *CsiError {
	sleepTime := 2 * time.Second

	for {
		time.Sleep(sleepTime)

		jobStatus, csiErr := TNSGetJobStatus(client, *jobID)
		if csiErr != nil {
			return csiErr
		}

		switch jobStatus.State {
		case "RUNNING":
			continue
		case "SUCCESS":
			return nil
		case "FAILED", "ABORTED":
			klog.Errorf("Job failed or aborted: %v", *jobStatus)
			return NewCsiError(codes.Internal, fmt.Errorf("%s", jobStatus.Err))
		default:
			klog.Errorf("default: %s", csiErr)
			return NewCsiError(codes.Internal, fmt.Errorf("Unexpected job state: %s", jobStatus.State))
		}
	}
}
