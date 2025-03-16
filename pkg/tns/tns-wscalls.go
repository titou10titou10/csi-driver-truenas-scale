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

/*
Truenas Scale WS client
*/

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

// --------
// Datasets
// --------

func TNSDatasetCreate(client *Client, driverName string, dsName string, reqCapacity int64, parameters map[string]string) (*TNSDataset, *CsiError) {
	klog.V(2).Infof("### TNSDatasetCreate dsName: %s reqCapacity: %d parameters: %s", dsName, reqCapacity, parameters)
	defer klog.V(2).Info("### TNSDatasetCreate")

	params := []interface{}{
		map[string]interface{}{
			"name":     dsName,
			"refquota": reqCapacity,
			"type":     "FILESYSTEM",
			"comments": driverName,
		},
	}
	ds, err := callTS[TNSDataset](client, "pool.dataset.create", params)
	if err != nil {

		if customErr, ok := err.(CustomError); ok {
			reason := strings.ToLower(customErr.Reason)

			switch {
			case strings.Contains(reason, "should be greater than"):
				// [11] VALIDATION EAGAIN: [EINVAL] pool_dataset_create.refquota: Should be greater than or equal to 1073741824 or Should be 0
				return nil, NewCsiError(codes.InvalidArgument, err)

			case strings.Contains(reason, "already exists"):
				// [11] VALIDATION EAGAIN: [EINVAL] pool_dataset_create.name: Path xxx already exists
				return nil, NewCsiError(codes.AlreadyExists, err)

			default:
				csiErr := NewCsiError(codes.Internal, err)
				klog.Errorf("Dataset creation failed: %v", csiErr)
				return nil, csiErr
			}

		} else {
			return nil, NewCsiError(codes.Internal, err)
		}
	}

	klog.V(2).Infof("++ Dataset creation successful: %v", ds)
	return &ds, nil
}

func TNSDatasetSetPermissions(client *Client, dsMountPoint string, parameters map[string]string) *CsiError {
	klog.V(2).Infof("### TNSDatasetSetPermissions dsMountPoint: %s parameters: %s", dsMountPoint, parameters)
	defer klog.V(2).Info("### TNSDatasetSetPermissions")

	params := []interface{}{
		map[string]interface{}{
			"path": dsMountPoint,
		},
	}
	data := params[0].(map[string]interface{})
	if p, ok := parameters["dsPermissionsMode"]; ok && p != "" {
		data["mode"] = p
	}
	if p, ok := parameters["dsPermissionsUser"]; ok && p != "" {
		if num, err := strconv.Atoi(p); err == nil {
			data["uid"] = num
		}
	}
	if p, ok := parameters["dsPermissionsGroup"]; ok && p != "" {
		if num, err := strconv.Atoi(p); err == nil {
			data["gid"] = num
		}
	}
	res, err := callTS[uint32](client, "filesystem.setperm", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Dataset Set permission failed: %s", csiErr)
		return csiErr
	}

	klog.V(2).Infof("++ Set permission OK: %d", res)
	return nil
}

func TNSDatasetGetPermissions(client *Client, dsMountPoint string) (*TNSDatasetStats, *CsiError) {
	klog.V(2).Infof("### TNSDatasetGetPermissions dsMountPoint: %s", dsMountPoint)
	defer klog.V(2).Info("### TNSDatasetGetPermissions")

	params := []interface{}{
		dsMountPoint,
	}
	res, err := callTS[TNSDatasetStats](client, "filesystem.stat", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Dataset Get permission failed: %s", csiErr)
		return nil, csiErr
	}

	klog.V(2).Infof("++ Get permission OK: %d", res)
	return &res, nil
}

func TNSDatasetSetSize(client *Client, dsName string, newSize int64) (*TNSDataset, *CsiError) {
	klog.V(2).Infof("### TNSDatasetSetSize dsName: %s newSize: %d", dsName, newSize)
	defer klog.V(2).Info("### TNSDatasetSetSize")

	params := []interface{}{
		dsName,
		map[string]interface{}{
			"refquota": newSize,
		},
	}

	res, err := callTS[TNSDataset](client, "pool.dataset.update", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Dataset Update Size failed: %v", csiErr)
		return nil, csiErr
	}

	klog.V(3).Infof("++ Dataset Update Size OK: %v", res)
	return &res, nil
}

func TNSDatasetPromote(client *Client, dsName string) *CsiError {
	klog.V(2).Infof("### TNSDatasetPromote dsName: %s", dsName)
	defer klog.V(2).Info("### TNSDatasetPromote")

	params := []interface{}{
		dsName,
	}
	res, err := callTS[bool](client, "pool.dataset.promote", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Dataset Promote failed: %v", csiErr)
		return csiErr
	}

	klog.V(3).Infof("++ Dataset Promote OK: %t", res)
	return nil
}

func TNSDatasetDelete(client *Client, dsName string) *CsiError {
	klog.V(2).Infof("### TNSDatasetDelete dsName: %s", dsName)
	defer klog.V(2).Info("### TNSDatasetDelete")

	params := []interface{}{
		dsName,
	}
	res, err := callTS[bool](client, "pool.dataset.delete", params)
	if err != nil {

		if customErr, ok := err.(CustomError); ok {
			reason := strings.ToLower(customErr.Reason)

			switch {
			case strings.Contains(reason, "filesystem has children"):
				// [14]  EFAULT: [EFAULT] Failed to delete dataset: cannot destroy 'xxxxxxxxx': filesystem has children
				csiErr := NewCsiError(codes.FailedPrecondition, err)
				klog.Errorf("++ Filesystem has children: %s", csiErr)
				return csiErr

			case strings.Contains(reason, "does not exist"):
				// [2] VALIDATION ENOENT: [ENOENT] None: PoolDataset xxxx does not exist
				klog.Warningf("++ Dataset does not exist, continue. %v", customErr.Reason)
				return nil

			default:
				csiErr := NewCsiError(codes.Internal, err)
				klog.Errorf("Dataset deletion failed: %v", csiErr)
				return csiErr
			}

		} else {
			return NewCsiError(codes.Internal, err)
		}
	}

	klog.V(3).Infof("++ Dataset Delete OK: %t", res)
	return nil
}

func TNSDatasetGet(client *Client, dsName string) (*TNSDataset, *CsiError) {
	klog.V(2).Infof("### TNSDatasetGet dsName: %s", dsName)
	defer klog.V(2).Info("### TNSDatasetGet")

	params := []interface{}{
		dsName,
	}
	res, err := callTS[TNSDataset](client, "pool.dataset.get_instance", params)
	if err != nil {

		if customErr, ok := err.(CustomError); ok {
			reason := strings.ToLower(customErr.Reason)

			switch {
			case strings.Contains(reason, "does not exist"):
				// [2] VALIDATION ENOENT: [ENOENT] None: PoolDataset xxxx does not exist
				klog.Errorf("++ Dataset does not exist. %v", customErr.Reason)
				return nil, NewCsiError(codes.Internal, err)

			default:
				csiErr := NewCsiError(codes.Internal, err)
				klog.Errorf("Dataset Get failed: %v", csiErr)
				return nil, csiErr
			}

		} else {
			return nil, NewCsiError(codes.Internal, err)
		}
	}

	klog.V(3).Info("++ Dataset Get OK")
	return &res, nil
}

// func TNSDatasetClone(client *Client, srcDsName string, dstDsName string) *CsiError {
// 	klog.V(2).Infof("### TNSDatasetClone srcdsName: %s dstdsName: %s", srcDsName, dstDsName)
// 	defer klog.V(2).Info("### TNSDatasetClone")

// 	params := []interface{}{
// 		map[string]interface{}{
// 			//"name":             "toto",
// 			"direction":        "PUSH",
// 			"transport":        "LOCAL",
// 			"source_datasets":  []interface{}{srcDsName},
// 			"target_dataset":   dstDsName,
// 			"recursive":        true,
// 			"auto":             false,
// 			"retention_policy": "NONE",
// 		},
// 	}
// 	res, err := callTS[bool](client, "replication.create", params)
// 	if err != nil {

// 		if customErr, ok := err.(CustomError); ok {
// 			reason := strings.ToLower(customErr.Reason)

// 			switch {
// 			case strings.Contains(reason, "does not exist"):
// 				// [2] VALIDATION ENOENT: [ENOENT] None: PoolDataset xxxx does not exist
// 				klog.Warningf("++ Dataset does not exist, continue. %v", customErr.Reason)
// 				return nil

// 			default:
// 				csiErr := NewCsiError(codes.Internal, err)
// 				klog.Errorf("Dataset deletion failed: %v", csiErr)
// 				return csiErr
// 			}

// 		} else {
// 			return NewCsiError(codes.Internal, err)
// 		}
// 	}

// 	klog.V(3).Infof("++ Dataset Clone OK: %t", res)
// 	return nil
// }

// --------
// Snapshots
// --------

func TNSSnapshotCreate(client *Client, dsName string, snapshotName string) (*TNSSnapshot, *CsiError) {
	klog.V(2).Infof("### TNSSnapshotCreate dsName: %s snapshotName: %s", dsName, snapshotName)
	defer klog.V(2).Info("### TNSSnapshotCreate")

	params := []interface{}{
		map[string]interface{}{
			"dataset": dsName,
			"name":    snapshotName,
		},
	}
	res, err := callTS[TNSSnapshot](client, "zfs.snapshot.create", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Snapshot Creation failed: %v", csiErr)
		return nil, csiErr
	}
	klog.V(2).Infof("++ Snapshot Creation OK: %v", res)
	return &res, nil
}

func TNSSnapshotDelete(client *Client, snapshotName string) (*bool, *CsiError) {
	klog.V(2).Infof("### TNSSnapshotCreate snapshotName: %s", snapshotName)
	defer klog.V(2).Info("### TNSSnapshotDelete")

	params := []interface{}{
		snapshotName,
	}
	res, err := callTS[bool](client, "zfs.snapshot.delete", params)
	if err != nil {

		if customErr, ok := err.(CustomError); ok {
			reason := strings.ToLower(customErr.Reason)

			switch {
			case strings.Contains(reason, "not found"):
				// [2] VALIDATION ENOENT: [ENOENT] None: Snapshot xxx not found
				klog.Warningf("++ Snapshot does not exist, continue. %v", customErr.Reason)
				res = true
				return &res, nil
			default:
				csiErr := NewCsiError(codes.Internal, err)
				klog.Errorf("Snapshot Delete failed: %v", csiErr)
				return nil, csiErr
			}

		} else {
			csiErr := NewCsiError(codes.Internal, err)
			klog.Errorf("Snapshot Delete failed: %v", csiErr)
			return nil, csiErr
		}
	}

	klog.V(2).Infof("++ Snapshot Delete OK: %v", res)
	return &res, nil
}

func TNSDatasetDestroySnapshotsJob(client *Client, dsName string) (*int, *CsiError) {
	klog.V(2).Infof("### TNSDatasetDestroySnapshotsJob dsName: %s", dsName)
	defer klog.V(2).Info("### TNSDatasetDestroySnapshotsJob")

	params := []interface{}{
		dsName,
	}

	jobID, err := callTS[int](client, "pool.dataset.destroy_snapshots", params)
	if err != nil {
		return nil, NewCsiError(codes.Internal, err)
	}

	klog.Infof("Job delete snapshots OK: %d", jobID)
	return &jobID, nil
}
func TNSSnapshotClone(client *Client, sourceSnapshotName string, targetDsName string) *CsiError {
	klog.V(2).Infof("### TNSSnapshotClone sourceSnapshotName: %s targetDsName: %s", sourceSnapshotName, targetDsName)
	defer klog.V(2).Info("### TNSSnapshotClone")

	params := []interface{}{
		map[string]interface{}{
			"snapshot":    sourceSnapshotName,
			"dataset_dst": targetDsName,
		},
	}
	res, err := callTS[bool](client, "zfs.snapshot.clone", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Snapshot Clone failed: %v", csiErr)
		return csiErr
	}
	klog.V(2).Infof("++ Snapshot Clone OK: %t", res)
	return nil
}

func TNSSnapshotGet(client *Client, dsName string) ([]TNSSnapshot, *CsiError) {
	klog.V(2).Infof("### TNSSnapshotGet dsName: %s", dsName)
	defer klog.V(2).Info("### TNSSnapshotGet")

	params := []interface{}{
		dsName, // The dataset name
		map[string]interface{}{
			"select": map[string]bool{"name": true}, // Correct selection format
		},
	}

	snapshots, err := callTS[[]TNSSnapshot](client, "zfs.snapshot.query", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("Snapshot Get failed: %v", csiErr)
		return nil, csiErr
	}

	return snapshots, nil
}

// ---------
// NFS Share
// ---------

func TNSShareNfsCreate(client *Client, dsMountPoint string, parameters map[string]string) (*string, *CsiError) {
	klog.V(2).Infof("### TNSShareNfsCreate dsMountPoint: %s parameters: %s", dsMountPoint, parameters)
	defer klog.V(2).Info("### TNSShareNfsCreate")

	params := []interface{}{
		map[string]interface{}{
			"path": dsMountPoint,
		},
	}
	data := params[0].(map[string]interface{})
	if p, ok := parameters["shareMaprootUser"]; ok && p != "" {
		data["maproot_user"] = p
	}
	if p, ok := parameters["shareMaprootGroup"]; ok && p != "" {
		data["maproot_group"] = p
	}
	if p, ok := parameters["shareMapallUser"]; ok && p != "" {
		data["mapall_user"] = p
	}
	if p, ok := parameters["shareMapallGroup"]; ok && p != "" {
		data["mapall_group"] = p
	}

	if p, ok := parameters["shareAllowedNetworks"]; ok && p != "" {
		ips := strings.Split(p, ",")
		var result []string
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			result = append(result, ip)
		}
		data["networks"] = result
	}
	if p, ok := parameters["shareAllowedHosts"]; ok && p != "" {
		data["hosts"] = strings.Split(p, ",")
	}

	nfs, err := callTS[TNSNFSShare](client, "sharing.nfs.create", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("NFS Share Create failed: %s", csiErr)
		return nil, csiErr
	}

	klog.V(3).Infof("++ NFS Share create OK: %v", nfs)
	return &nfs.Path, nil
}

func TNSShareNfsGet(client *Client, mountPoint string) (*TNSNFSShare, *CsiError) {
	klog.V(2).Infof("### TNSShareNfsGet dsName: %s", mountPoint)
	defer klog.V(2).Info("### TNSShareNfsGet")

	params := []interface{}{
		[]interface{}{
			[]interface{}{"path", "=", mountPoint},
		},
	}

	shares, err := callTS[[]TNSNFSShare](client, "sharing.nfs.query", params)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("NFS Share Get failed: %s", csiErr)
		return nil, csiErr
	}
	klog.V(3).Info("++ NFS Share Get OK")
	if len(shares) == 0 {
		return nil, nil
	} else {
		return &shares[0], nil
	}
}

// -----
// Other
// -----

func TNSOneTimeReplicationJob(client *Client, srcDsName, snapshotName string, destDsName string) (*int, *CsiError) {
	klog.V(2).Infof("### TNSOneTimeReplicationJob srcDsName: %s snapshotName: %s destDsName: %s", srcDsName, snapshotName, destDsName)
	defer klog.V(2).Info("### TNSOneTimeReplicationJob")

	params := []interface{}{
		map[string]interface{}{
			"direction":        "PUSH",
			"transport":        "LOCAL",
			"source_datasets":  []string{srcDsName},
			"target_dataset":   destDsName,
			"recursive":        false,
			"retention_policy": "NONE",
			"readonly":         "IGNORE",
			"properties":       false,
			"name_regex":       snapshotName,
		},
	}

	jobID, err := callTS[int](client, "replication.run_onetime", params)
	if err != nil {
		return nil, NewCsiError(codes.Internal, err)
	}

	return &jobID, nil
}

func TNSGetJobStatus(client *Client, jobID int) (*TNSJobStatus, *CsiError) {
	klog.V(2).Infof("### TNSGetJobStatus jobID: %d", jobID)
	defer klog.V(2).Info("### TNSGetJobStatus")

	params := []interface{}{
		[][]interface{}{
			{"id", "=", jobID},
		},
	}
	jobs, err := callTS[[]TNSJobStatus](client, "core.get_jobs", params)
	if err != nil {
		return nil, NewCsiError(codes.Internal, err)
	}

	if len(jobs) == 0 {
		return nil, NewCsiError(codes.Internal, errors.New("job not found"))
	}

	return &jobs[0], nil
}

// ----------------------
// Calls to Truenas Scale
// ----------------------

func callTS[T any](c *Client, method string, params interface{}) (T, error) {

	var result T

	request := WSRequest{
		ID:     uuid.New().String(),
		Method: method,
		Params: params,
	}
	if c.legacyTns {
		request.Msg = "method"
	} else {
		request.JsonRPC = "2.0"
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		klog.Errorf("Failed to encode JSON: %v", err)
		return result, err
	}
	if request.Method != "auth.login_with_api_key" {
		// Do not log apiKey
		klog.V(2).Infof("S: %s", jsonData)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		klog.Errorf("Failed to send message: %v", err)
		return result, err
	}

	klog.V(3).Infof("Message sent, waiting for response...")
	_, response, err := c.conn.ReadMessage()
	if err != nil {
		klog.Errorf("Failed to read response: %v", err)
		return result, err
	}

	// Log on single line (maybe truncated..) or split response in lines
	if klog.V(3).Enabled() {
		logChunks(string(response))
	} else {
		klog.V(2).Infof("R: %s", string(response))
	}

	var wsResp WSResponse
	err = json.Unmarshal(response, &wsResp)
	if err != nil {
		klog.Errorf("Failed to decode JSON response: %v", err)
		return result, err
	}

	// Error server side
	if wsResp.Error.IsErrorPresent() {
		e := wsResp.Error.ToError()
		klog.Errorf("Error from truenas: %v", e)
		return result, e
	}

	if err := json.Unmarshal(wsResp.Result, &result); err != nil {
		klog.Errorf("Failed to decode Result field: %v", err)
		return result, err

	}

	return result, nil
}

func TNSLogin(legacyTns bool, conn *websocket.Conn, apiKey string) *CsiError {
	klog.V(2).Infof("### TNSLogin legacyTns? %t", legacyTns)
	defer klog.V(2).Info("### TNSLogin")

	// Build temp client
	tempClient := &Client{
		legacyTns: legacyTns,
		conn:      conn,
	}

	// Login with API key
	params := []interface{}{
		apiKey,
	}
	res, err := callTS[bool](tempClient, "auth.login_with_api_key", params)
	if err != nil {
		csiError := NewCsiError(codes.Unauthenticated, err)
		klog.Errorf("Login failed: %s", csiError)
		return csiError
	}
	klog.V(3).Infof("Login OK: %t", res)

	return nil
}

// ---------------------------------------
// Utils
// ---------------------------------------

const chunkSize = 900

func splitIntoChunks(s string, chunkSize int) []string {
	var chunks []string
	for len(s) > 0 {
		if len(s) > chunkSize {
			chunks = append(chunks, s[:chunkSize])
			s = s[chunkSize:]
		} else {
			chunks = append(chunks, s)
			s = ""
		}
	}
	return chunks
}

func logChunks(s string) {
	chunks := splitIntoChunks(s, chunkSize)
	for _, chunk := range chunks {
		klog.V(3).Infof("R: %s", chunk)
	}
}
