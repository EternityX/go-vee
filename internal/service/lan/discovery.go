// Please refer to the following guide for more information:
// https://app-h5.govee.com/user-manual/wlan-guide

package lan

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	multicastAddr = "239.255.255.250:4001"
	listenPort    = "4002"
)

type ScanRequest struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			AccountTopic string `json:"account_topic"`
		} `json:"data"`
	} `json:"msg"`
}

type ScanResponse struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			IP              string `json:"ip"`
			Device          string `json:"device"`
			SKU             string `json:"sku"`
			BleVersionHard  string `json:"bleVersionHard"`
			BleVersionSoft  string `json:"bleVersionSoft"`
			WifiVersionHard string `json:"wifiVersionHard"`
			WifiVersionSoft string `json:"wifiVersionSoft"`
		} `json:"data"`
	} `json:"msg"`
}

// Scans for Govee devices on the local network
func DiscoverDevices(timeout time.Duration) ([]ScanResponse, error) {
	// Create UDP address for multicast
	multicastAddr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	// Create UDP connection for sending
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	// Create UDP server for receiving responses
	serverAddr, err := net.ResolveUDPAddr("udp", ":"+listenPort)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	server, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP server: %w", err)
	}
	defer server.Close()

	// Prepare scan request
	scanReq := ScanRequest{}
	scanReq.Msg.Cmd = "scan"
	scanReq.Msg.Data.AccountTopic = "reserve"

	reqData, err := json.Marshal(scanReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scan request: %w", err)
	}

	// Send scan request
	_, err = conn.WriteToUDP(reqData, multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send scan request: %w", err)
	}

	// Collect responses
	var devices []ScanResponse
	deadline := time.Now().Add(timeout)
	buffer := make([]byte, 1024)

	for time.Now().Before(deadline) {
		server.SetReadDeadline(deadline)
		n, _, err := server.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			log.Printf("Error reading UDP response: %v", err)
			continue
		}

		var resp ScanResponse
		if err := json.Unmarshal(buffer[:n], &resp); err != nil {
			log.Printf("Error unmarshaling response: %v", err)
			continue
		}

		devices = append(devices, resp)
	}

	return devices, nil
}
