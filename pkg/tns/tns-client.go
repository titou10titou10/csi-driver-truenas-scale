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
	"crypto/tls"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

const legacyPath = "/websocket"
const modernPath = "/api/current"
const timeout = 10

type Client struct {
	legacyTns  bool // true for Truenas Scale v24.10-: endpoint="/websocket". false for Truenas Scale v25.04+: endpoint="/api/current" + jsonrcp
	conn       *websocket.Conn
	mu         sync.Mutex
	lastActive time.Time
	inUse      bool
}

type ConnectionPool struct {
	mu    sync.Mutex
	conns map[string][]*Client
}

var pool = &ConnectionPool{
	conns: make(map[string][]*Client),
}

func GetClient(tnsWsUrl, apiKey string, insecureSkipVerify bool) (*Client, *CsiError) {
	klog.V(3).Infof("GetClient tnsWsUrl: %s insecureSkipVerify? %t", tnsWsUrl, insecureSkipVerify)

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if clients, exists := pool.conns[tnsWsUrl]; exists {
		for _, client := range clients {
			client.mu.Lock()
			if !client.inUse && client.isAlive() { // Check if the connection is still alive
				client.inUse = true
				client.mu.Unlock()
				klog.V(2).Infof("Reusing WebSocket connection for %s", tnsWsUrl)
				return client, nil
			}
			client.mu.Unlock()
		}
	}

	klog.V(2).Infof("Creating new WebSocket connection for %s", tnsWsUrl)
	client, err := newClient(tnsWsUrl, apiKey, insecureSkipVerify)
	if err != nil {
		return nil, err
	}

	client.inUse = true
	pool.conns[tnsWsUrl] = append(pool.conns[tnsWsUrl], client)
	return client, nil
}

func ReleaseClient(client *Client) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.inUse = false

	if client.isAlive() {
		klog.V(3).Infof("Released WebSocket connection back to the pool")
	} else {
		// If the connection is dead, close it
		client.conn.Close()
		klog.V(2).Infof("Closed dead WebSocket connection")
	}
}

func newClient(tnsWsUrl string, apiKey string, insecureSkipVerify bool) (*Client, *CsiError) {
	klog.V(3).Infof("newClient tnsWsUrl: %s insecureSkipVerify? %t", tnsWsUrl, insecureSkipVerify)

	// Truenas Scale < v25.0
	// ws://<truenas.server>/websocket
	// wss://<truenas.server>/websocket
	// Truenas Scale >= v25.0
	// ws://<truenas.server>/api/current
	// wss://<truenas.server>/api/current

	parsedURL, err := url.Parse(tnsWsUrl)
	if err != nil {
		csiErr := NewCsiError(codes.InvalidArgument, err)
		klog.Errorf("invalid truenas scale url: %s", csiErr)
		return nil, csiErr
	}
	scheme := parsedURL.Scheme
	if scheme != "ws" && scheme != "wss" {
		csiErr := NewCsiError(codes.InvalidArgument, err)
		klog.Errorf("invalid truenas scale url scheme. must be either 'ws' or 'wss': %s", csiErr)
		return nil, csiErr
	}
	path := parsedURL.Path
	if path != legacyPath && path != modernPath {
		csiErr := NewCsiError(codes.InvalidArgument, err)
		klog.Errorf("invalid truenas scale url path. must be either '/websocket (v24.10-)' or '/api/current (v25.04+)': %s", csiErr)
		return nil, csiErr
	}

	// TLS or not
	var dialer websocket.Dialer
	if scheme == "wss" {
		dialer = websocket.Dialer{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
			HandshakeTimeout: timeout * time.Second,
		}
	} else {
		dialer = websocket.Dialer{
			HandshakeTimeout: timeout * time.Second,
		}
	}
	legacyTns := true
	if path == modernPath {
		legacyTns = false
	}

	klog.V(3).Infof("tnsWsUrl: %s", tnsWsUrl)

	// Perform a WebSocket connection
	conn, _, err := dialer.Dial(tnsWsUrl, nil)
	if err != nil {
		csiErr := NewCsiError(codes.Internal, err)
		klog.Errorf("WebSocket connection failed: %s", csiErr)
		return nil, csiErr
	}
	klog.V(3).Infof("WebSocket connection established with %s", tnsWsUrl)

	// Send WebSocket "connect" message
	if legacyTns {
		jsonMessage := []byte(`{"msg": "connect", "version": "1", "support": ["1" ]}`)
		if err := conn.WriteMessage(websocket.TextMessage, jsonMessage); err != nil {
			csiErr := NewCsiError(codes.Internal, err)
			klog.Errorf("Failed to send connect message: %s", csiErr)
			conn.Close()
			return nil, csiErr
		}
		_, response, err := conn.ReadMessage()
		if err != nil {
			csiErr := NewCsiError(codes.Internal, err)
			klog.Errorf("Failed to read connect response: %s", csiErr)
			conn.Close()
			return nil, csiErr
		}
		klog.V(3).Infof("Truenas Connect OK: %s", string(response))
	}

	// Login
	csiErr := TNSLogin(legacyTns, conn, apiKey)
	if csiErr != nil {
		conn.Close()
		return nil, csiErr
	}

	return &Client{
		legacyTns: legacyTns,
		conn:      conn,
		inUse:     true,
	}, nil
}

func (client *Client) isAlive() bool {
	klog.V(4).Infof("isAlive?")

	if client.conn == nil {
		return false
	}

	client.lastActive = time.Now()

	err := client.conn.WriteMessage(websocket.PingMessage, nil)
	if err != nil {
		klog.Errorf("Ping failed for connection: %s. isAlive: No", err)
		return false // Connection is dead or unreachable
	}

	// set a timeout for a Pong response using ReadMessage or a dedicated goroutine?
	klog.V(3).Infof("isAlive: Yes")
	return true
}

// TNSStartWSSCleanupRoutine(10*time.Minute, 10*time.Minute) // Close inactive connections after 10 minutes, check every minute

func TNSStartWSSCleanupRoutine(maxIdleTime time.Duration, interval time.Duration) {
	klog.V(2).Infof("WSS transactions will be checked for cleaning every %s. Connections inactive since %s will be closed", interval, maxIdleTime)

	go func() {
		for {
			time.Sleep(interval)
			cleanupInactiveConnections(maxIdleTime)
		}
	}()
}

func cleanupInactiveConnections(maxIdleTime time.Duration) {
	klog.V(3).Info("Check wss connections for cleanup")

	pool.mu.Lock()
	defer pool.mu.Unlock()

	for host, clients := range pool.conns {
		activeClients := []*Client{}
		for _, client := range clients {
			client.mu.Lock()

			if time.Since(client.lastActive) > maxIdleTime {
				klog.V(2).Infof("Closing inactive WebSocket connection for %s", host)
				client.conn.Close()
			} else {
				activeClients = append(activeClients, client) // Keep active clients
			}

			client.mu.Unlock()
		}
		pool.conns[host] = activeClients
	}
}
