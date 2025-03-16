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
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
)

// -------------------------
// truenas components
// -------------------------

type TNSJobStatus struct {
	ID     int         `json:"id"`
	State  string      `json:"state"`
	Result interface{} `json:"result,omitempty"`
	Err    interface{} `json:"error,omitempty"`
}

type TNSSnapshot struct {
	Id           string                `json:"id,omitempty"`            // dataset@snapshot_name
	Name         string                `json:"name,omitempty"`          // dataset@snapshot_name
	SnapshotName string                `json:"snapshot_name,omitempty"` // Short name
	Dataset      string                `json:"dataset,omitempty"`       // full dsname
	Pool         string                `json:"pool,omitempty"`          // "SNAPSHOT"
	Type         string                `json:"type,omitempty"`
	Properties   TNSSnapshotProperties `json:"properties,omitempty"`
}

type TNSSnapshotProperties struct {
	// Compressratio     ZFSProperty `json:"compressratio,omitempty"`
	// Createtxg         ZFSProperty `json:"createtxg,omitempty"`
	// Creation          ZFSProperty `json:"creation,omitempty"`
	// DeferDestroy      ZFSProperty `json:"defer_destroy,omitempty"`
	// Encryptionroot    ZFSProperty `json:"encryptionroot,omitempty"`
	// GUID              ZFSProperty `json:"guid,omitempty"`
	// Inconsistent      ZFSProperty `json:"inconsistent,omitempty"`
	// IVSetGUID         ZFSProperty `json:"ivsetguid,omitempty"`
	// KeyGUID           ZFSProperty `json:"keyguid,omitempty"`
	// KeyStatus         ZFSProperty `json:"keystatus,omitempty"`
	// LogicalReferenced ZFSProperty `json:"logicalreferenced,omitempty"`
	// Name              ZFSProperty `json:"name,omitempty"`
	// NumClones         ZFSProperty `json:"numclones,omitempty"`
	// ObjSetID          ZFSProperty `json:"objsetid,omitempty"`
	// RedactSnaps       ZFSProperty `json:"redact_snaps,omitempty"`
	// Redacted          ZFSProperty `json:"redacted,omitempty"`
	// RefCompressRatio  ZFSProperty `json:"refcompressratio,omitempty"`
	Referenced ZFSProperty `json:"referenced,omitempty"`
	// RemapTXG          ZFSProperty `json:"remaptxg,omitempty"`
	// Type              ZFSProperty `json:"type,omitempty"`
	// Unique            ZFSProperty `json:"unique,omitempty"`
	// Used              ZFSProperty `json:"used,omitempty"`
	// UserAccounting    ZFSProperty `json:"useraccounting,omitempty"`
	// UserRefs          ZFSProperty `json:"userrefs,omitempty"`
	// Written           ZFSProperty `json:"written,omitempty"`
	// ACLType           ZFSProperty `json:"acltype,omitempty"`
	// CaseSensitivity   ZFSProperty `json:"casesensitivity,omitempty"`
	// Context           ZFSProperty `json:"context,omitempty"`
	// DefContext        ZFSProperty `json:"defcontext,omitempty"`
	// Devices           ZFSProperty `json:"devices,omitempty"`
	// Encryption        ZFSProperty `json:"encryption,omitempty"`
	// Exec              ZFSProperty `json:"exec,omitempty"`
	// FSContext         ZFSProperty `json:"fscontext,omitempty"`
	// MLSLabel          ZFSProperty `json:"mlslabel,omitempty"`
	// NBMand            ZFSProperty `json:"nbmand,omitempty"`
	// Normalization     ZFSProperty `json:"normalization,omitempty"`
	// Prefetch          ZFSProperty `json:"prefetch,omitempty"`
	// PrimaryCache      ZFSProperty `json:"primarycache,omitempty"`
	// RootContext       ZFSProperty `json:"rootcontext,omitempty"`
	// SecondaryCache    ZFSProperty `json:"secondarycache,omitempty"`
	// SetUID            ZFSProperty `json:"setuid,omitempty"`
	// UTF8Only          ZFSProperty `json:"utf8only,omitempty"`
	// Version           ZFSProperty `json:"version,omitempty"`
	// VolSize           ZFSProperty `json:"volsize,omitempty"`
	// XAttr             ZFSProperty `json:"xattr,omitempty"`
}

type TNSDataset struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Pool       string      `json:"pool,omitempty"`
	Available  ZFSProperty `json:"available,omitempty"`
	Comments   ZFSProperty `json:"comments,omitempty"`
	MountPoint string      `json:"mountpoint,omitempty"`
	RefQuota   ZFSProperty `json:"refquota,omitempty"`

	// Type           string      `json:"type,omitempty"`
	// Encrypted      bool        `json:"encrypted,omitempty"`
	// EncryptionRoot string      `json:"encryption_root,omitempty"`
	// KeyLoaded      bool        `json:"key_loaded,omitempty"`
	// Children       []any  `json:"children"`
	// ManagedBy      ZFSProperty `json:"managedby,omitempty"`
	// Deduplication  ZFSProperty `json:"deduplication,omitempty"`
	// ACLMode        ZFSProperty `json:"aclmode"`
	// ACLType        ZFSProperty `json:"acltype"`

	// ATime                 ZFSProperty `json:"atime"`
	// CaseSensitivity       ZFSProperty `json:"casesensitivity"`
	// Checksum              ZFSProperty `json:"checksum"`
	// Compression           ZFSProperty `json:"compression"`
	// CompressRatio         ZFSProperty `json:"compressratio"`
	// Copies                ZFSProperty `json:"copies"`
	// VolSize               ZFSProperty `json:"volsize,omitempty"`
	// VolBlockSize          ZFSProperty `json:"volblocksize,omitempty"`
	// Sparse                ZFSProperty `json:"sparse,omitempty"`
	// ForceSize             ZFSProperty `json:"force_size,omitempty"`
	// Sync                  ZFSProperty `json:"sync,omitempty"`
	// SnapDev               ZFSProperty `json:"snapdev,omitempty"`
	// Exec                  ZFSProperty `json:"exec,omitempty"`
	// Quota                 ZFSProperty `json:"quota,omitempty"`
	// QuotaWarning          ZFSProperty `json:"quota_warning,omitempty"`
	// QuotaCritical         ZFSProperty `json:"quota_critical,omitempty"`
	// RefQuotaWarning       ZFSProperty `json:"refquota_warning,omitempty"`
	// RefQuotaCritical      ZFSProperty `json:"refquota_critical,omitempty"`
	// Reservation           ZFSProperty `json:"reservation,omitempty"`
	// RefReservation        ZFSProperty `json:"refreservation,omitempty"`
	// SpecialSmallBlockSize ZFSProperty `json:"special_small_block_size,omitempty"`
	// SnapDir               ZFSProperty `json:"snapdir,omitempty"`
	// ReadOnly              ZFSProperty `json:"readonly,omitempty"`
	// RecordSize            ZFSProperty `json:"recordsize,omitempty"`
	// EncryptionOptions     ZFSProperty `json:"encryption_options"`
	// Encryption            ZFSProperty `json:"encryption"`
	// InheritEncryption     ZFSProperty `json:"inherit_encryption"`
}

type TNSNFSShare struct {
	Path         string   `json:"path"`                    // Required
	Aliases      []string `json:"aliases,omitempty"`       // Default: []
	Comment      string   `json:"comment,omitempty"`       // Default: ""
	Networks     []string `json:"networks,omitempty"`      // Default: []
	Hosts        []string `json:"hosts,omitempty"`         // Default: []
	ReadOnly     bool     `json:"ro,omitempty"`            // Default: false
	MapRootUser  string   `json:"maproot_user,omitempty"`  // Default: null
	MapRootGroup string   `json:"maproot_group,omitempty"` // Default: null
	MapAllUser   string   `json:"mapall_user,omitempty"`   // Default: null
	MapAllGroup  string   `json:"mapall_group,omitempty"`  // Default: null
	Security     []string `json:"security,omitempty"`      // Default: []
	Enabled      bool     `json:"enabled,omitempty"`       // Default: true
	ID           uint     `json:"id,omitempty"`            // Optional
	Locked       bool     `json:"locked,omitempty"`        // Optional
}

type TNSDatasetStats struct {
	//Realpath        string   `json:"realpath"`
	//Size            int      `json:"size"`
	//AllocationSize  int      `json:"allocation_size"`
	Mode int `json:"mode"`
	UID  int `json:"uid"`
	GID  int `json:"gid"`
	//Atime           float64  `json:"atime"`
	//Mtime           float64  `json:"mtime"`
	//Ctime           float64  `json:"ctime"`
	//Btime           float64  `json:"btime"`
	//Dev             int      `json:"dev"`
	//MountID int `json:"mount_id"`
	//Inode           int      `json:"inode"`
	//Nlink        int      `json:"nlink"`
	//IsMountpoint bool     `json:"is_mountpoint"`
	//IsCtldir     bool     `json:"is_ctldir"`
	//Attributes   []string `json:"attributes"`
	//User         *string  `json:"user"`
	//Group        *string  `json:"group"`
	//Acl             bool     `json:"acl"`
}

type EncryptionOptions struct {
	GenerateKey bool    `json:"generate_key"`
	PBKDF2Iters int     `json:"pbkdf2iters"`
	Algorithm   string  `json:"algorithm"`
	Passphrase  *string `json:"passphrase,omitempty"`
	Key         *string `json:"key,omitempty"`
}

type ZFSProperty struct {
	Parsed     any     `json:"parsed"`
	RawValue   string  `json:"rawvalue"`
	Source     string  `json:"source"`
	SourceInfo *string `json:"source_info,omitempty"`
	Value      string  `json:"value"`
}

// -------------------------
// truenas WS request
// -------------------------

type WSRequest struct {
	ID      string      `json:"id"`
	Msg     string      `json:"msg,omitempty"`     // Truenas Scale < 25.x
	JsonRPC string      `json:"jsonrpc,omitempty"` // Truenas Scale >= 25.x
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// -------------------------
// truenas WS result message
// -------------------------

type WSResponse struct {
	ID     string          `json:"id"`
	Msg    string          `json:"msg"`
	Error  Error           `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

type Error struct {
	Code    int    `json:"error,omitempty"`
	Errname string `json:"errname,omitempty"`
	Type    string `json:"type,omitempty"`
	Reason  string `json:"reason,omitempty"`
	// Trace   Trace  `json:"trace,omitempty"`
}

func (e Error) IsErrorPresent() bool {
	//	return !(e.Code == 0 && e.Errname == "" && e.Type == "" && e.Reason == "" && len(e.Trace.Frames) == 0)
	return !(e.Code == 0 && e.Errname == "" && e.Type == "" && e.Reason == "")
}
func (e Error) ToError() error {
	return CustomError{
		Code:    e.Code,
		Type:    e.Type,
		Errname: e.Errname,
		Reason:  e.Reason,
	}
}

// -------------------------------------
// Error Between CSI Methods and Backend
// -------------------------------------

type CsiError struct {
	Code codes.Code
	Err  error
}

func NewCsiError(code codes.Code, err error) *CsiError {
	return &CsiError{
		Code: code,
		Err:  err,
	}
}

func (e CsiError) String() string {
	errorStr := "<nil>"
	if e.Err != nil {
		errorStr = e.Err.Error()
	}

	return fmt.Sprintf("Code: %s, Error: %s", e.Code, errorStr)
}

func (e *CsiError) Error() string {
	return e.String()
}

func (e *CsiError) Is(target error) bool {
	if ce, ok := target.(*CsiError); ok {
		return e.Code == ce.Code
	}
	return false
}

// ------------------------------

type CustomError struct {
	Code    int
	Type    string
	Errname string
	Reason  string
}

func (e CustomError) Error() string {
	return fmt.Sprintf("[%d] %s %s: %s", e.Code, e.Type, e.Errname, e.Reason)
}

type Trace struct {
	Class  string  `json:"class,omitempty"`
	Frames []Frame `json:"frames,omitempty"`
}

type Frame struct {
	Filename string                 `json:"filename"`
	Lineno   int                    `json:"lineno"`
	Method   string                 `json:"method"`
	Line     string                 `json:"line"`
	Argspec  []string               `json:"argspec"`
	Locals   map[string]interface{} `json:"locals"`
}
