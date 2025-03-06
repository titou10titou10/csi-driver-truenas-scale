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
	"reflect"
	"testing"
	"time"
)

var (
	invalidEndpoint = "invalid-endpoint"
	emptyAddr       = "tcp://"
)

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		desc        string
		endpoint    string
		resproto    string
		respaddr    string
		expectedErr error
	}{
		{
			desc:        "invalid endpoint",
			endpoint:    invalidEndpoint,
			expectedErr: fmt.Errorf("Invalid endpoint: %v", invalidEndpoint),
		},
		{
			desc:        "empty address",
			endpoint:    emptyAddr,
			expectedErr: fmt.Errorf("Invalid endpoint: %v", emptyAddr),
		},
		{
			desc:        "valid tcp",
			endpoint:    "tcp://address",
			resproto:    "tcp",
			respaddr:    "address",
			expectedErr: nil,
		},
		{
			desc:        "valid unix",
			endpoint:    "unix://address",
			resproto:    "unix",
			respaddr:    "address",
			expectedErr: nil,
		},
	}

	for _, test := range cases {
		test := test //pin
		t.Run(test.desc, func(t *testing.T) {
			proto, addr, err := ParseEndpoint(test.endpoint)

			// Verify
			if test.expectedErr == nil && err != nil {
				t.Errorf("test %q failed: %v", test.desc, err)
			}
			if test.expectedErr != nil && err == nil {
				t.Errorf("test %q failed; expected error %v, got success", test.desc, test.expectedErr)
			}
			if test.expectedErr == nil {
				if test.resproto != proto {
					t.Errorf("test %q failed; expected proto %v, got proto %v", test.desc, test.resproto, proto)
				}
				if test.respaddr != addr {
					t.Errorf("test %q failed; expected addr %v, got addr %v", test.desc, test.respaddr, addr)
				}
			}
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		method string
		level  int32
	}{
		{
			method: "/csi.v1.Identity/Probe",
			level:  8,
		},
		{
			method: "/csi.v1.Node/NodeGetCapabilities",
			level:  8,
		},
		{
			method: "/csi.v1.Node/NodeGetVolumeStats",
			level:  8,
		},
		{
			method: "",
			level:  2,
		},
		{
			method: "unknown",
			level:  2,
		},
	}

	for _, test := range tests {
		level := getLogLevel(test.method)
		if level != test.level {
			t.Errorf("returned level: (%v), expected level: (%v)", level, test.level)
		}
	}
}

// getWorkDirPath returns the path to the current working directory

func TestGetServerFromSource(t *testing.T) {
	tests := []struct {
		desc     string
		tnsWsUrl string
		result   string
	}{
		{
			desc:     "ipv4",
			tnsWsUrl: "wss://10.127.0.1",
			result:   "10.127.0.1",
		},
		{
			desc:     "ipv6",
			tnsWsUrl: "wss://0:0:0:0:0:0:0:1",
			result:   "[0:0:0:0:0:0:0:1]",
		},
		{
			desc:     "ipv6 with brackets",
			tnsWsUrl: "wss://[0:0:0:0:0:0:0:2]",
			result:   "[0:0:0:0:0:0:0:2]",
		},
		{
			desc:     "other fqdn",
			tnsWsUrl: "wss://bing.com",
			result:   "bing.com",
		},
	}

	for _, test := range tests {
		result, _ := getServerFromSource(test.tnsWsUrl)
		if *result != test.result {
			t.Errorf("Unexpected result: %s, expected: %s", *result, test.result)
		}
	}
}

func TestValidateOnDeleteValue(t *testing.T) {
	tests := []struct {
		desc     string
		onDelete string
		expected error
	}{
		{
			desc:     "empty value",
			onDelete: "",
			expected: nil,
		},
		{
			desc:     "delete value",
			onDelete: "delete",
			expected: nil,
		},
		{
			desc:     "retain value",
			onDelete: "retain",
			expected: nil,
		},
		{
			desc:     "Retain value",
			onDelete: "Retain",
			expected: nil,
		},
		{
			desc:     "Delete value",
			onDelete: "Delete",
			expected: nil,
		},
		{
			desc:     "Archive value",
			onDelete: "Archive",
			expected: nil,
		},
		{
			desc:     "archive value",
			onDelete: "archive",
			expected: nil,
		},
		{
			desc:     "invalid value",
			onDelete: "invalid",
			expected: fmt.Errorf("invalid value %s for OnDelete, supported values are %v", "invalid", supportedOnDeleteValues),
		},
	}

	for _, test := range tests {
		result := validateOnDeleteValue(test.onDelete)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("test[%s]: unexpected output: %v, expected result: %v", test.desc, result, test.expected)
		}
	}
}

func TestWaitUntilTimeout(t *testing.T) {
	tests := []struct {
		desc        string
		timeout     time.Duration
		execFunc    ExecFunc
		timeoutFunc TimeoutFunc
		expectedErr error
	}{
		{
			desc:    "execFunc returns error",
			timeout: 1 * time.Second,
			execFunc: func() error {
				return fmt.Errorf("execFunc error")
			},
			timeoutFunc: func() error {
				return fmt.Errorf("timeout error")
			},
			expectedErr: fmt.Errorf("execFunc error"),
		},
		{
			desc:    "execFunc timeout",
			timeout: 1 * time.Second,
			execFunc: func() error {
				time.Sleep(2 * time.Second)
				return nil
			},
			timeoutFunc: func() error {
				return fmt.Errorf("timeout error")
			},
			expectedErr: fmt.Errorf("timeout error"),
		},
		{
			desc:    "execFunc completed successfully",
			timeout: 1 * time.Second,
			execFunc: func() error {
				return nil
			},
			timeoutFunc: func() error {
				return fmt.Errorf("timeout error")
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		err := WaitUntilTimeout(test.timeout, test.execFunc, test.timeoutFunc)
		if err != nil && (err.Error() != test.expectedErr.Error()) {
			t.Errorf("unexpected error: %v, expected error: %v", err, test.expectedErr)
		}
	}
}
