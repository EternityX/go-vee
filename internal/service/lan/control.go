// Please refer to the following guide for more information:
// https://app-h5.govee.com/user-manual/wlan-guide

package lan

import (
	"encoding/json"
	"fmt"
	"net"
)

func clampValue(value, min, max int) int {
	if value < min {
		return min
	}

	if value > max {
		return max
	}

	return value
}

type ControlRequest struct {
	Msg struct {
		Cmd  string      `json:"cmd"`
		Data interface{} `json:"data"`
	} `json:"msg"`
}

type ControlResponse struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			OnOff      int `json:"onOff,omitempty"`
			Brightness int `json:"brightness,omitempty"`
			Color      struct {
				R int `json:"r"`
				G int `json:"g"`
				B int `json:"b"`
			} `json:"color,omitempty"`
			ColorTemInKelvin int `json:"colorTemInKelvin,omitempty"`
		} `json:"data"`
	} `json:"msg"`
}

// Sends a control command to a device over LAN
func ControlDevice(deviceIP string, cmd string, data interface{}) error {
	addr, err := net.ResolveUDPAddr("udp", deviceIP+":4003")
	if err != nil {
		return fmt.Errorf("failed to resolve device address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}
	defer conn.Close()

	// Prepare control request
	req := ControlRequest{}
	req.Msg.Cmd = cmd
	req.Msg.Data = data

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal control request: %w", err)
	}

	// Send control request
	_, err = conn.Write(reqData)
	if err != nil {
		return fmt.Errorf("failed to send control request: %w", err)
	}

	// Read response if it's a status query
	if cmd == "devStatus" {
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		var resp ControlResponse
		if err := json.Unmarshal(buffer[:n], &resp); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Common control commands
func TurnOn(deviceIP string) error {
	data := struct {
		Value int `json:"value"`
	}{
		Value: 1,
	}

	return ControlDevice(deviceIP, "turn", data)
}

func TurnOff(deviceIP string) error {
	data := struct {
		Value int `json:"value"`
	}{
		Value: 0,
	}

	return ControlDevice(deviceIP, "turn", data)
}

func SetBrightness(deviceIP string, brightness int) error {
	brightness = clampValue(brightness, 1, 100)

	data := struct {
		Value int `json:"value"`
	}{
		Value: brightness,
	}

	return ControlDevice(deviceIP, "brightness", data)
}

func SetColor(deviceIP string, r, g, b int) error {
	r = clampValue(r, 0, 255)
	g = clampValue(g, 0, 255)
	b = clampValue(b, 0, 255)

	data := struct {
		Color struct {
			R int `json:"r"`
			G int `json:"g"`
			B int `json:"b"`
		} `json:"color"`
		ColorTemInKelvin int `json:"colorTemInKelvin"`
	}{
		Color: struct {
			R int `json:"r"`
			G int `json:"g"`
			B int `json:"b"`
		}{
			R: r,
			G: g,
			B: b,
		},
		ColorTemInKelvin: 0, // Set to 0 to use RGB values
	}

	return ControlDevice(deviceIP, "colorwc", data)
}

// Queries the status of a device over LAN
func GetDeviceStatus(deviceIP string) (*ControlResponse, error) {
	data := struct{}{} // Empty data for status query

	addr, err := net.ResolveUDPAddr("udp", deviceIP+":4003")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve device address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to device: %w", err)
	}
	defer conn.Close()

	// Prepare status request
	req := ControlRequest{}
	req.Msg.Cmd = "devStatus"
	req.Msg.Data = data

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status request: %w", err)
	}

	// Send status request
	_, err = conn.Write(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send status request: %w", err)
	}

	// Read response
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp ControlResponse
	if err := json.Unmarshal(buffer[:n], &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}
