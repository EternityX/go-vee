package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
)

const (
	DeviceTypeLight         = "devices.types.light"
	DeviceTypeAirPurifier   = "devices.types.air_purifier"
	DeviceTypeThermometer   = "devices.types.thermometer"
	DeviceTypeSocket        = "devices.types.socket"
	DeviceTypeSensor        = "devices.types.sensor"
	DeviceTypeHeater        = "devices.types.heater"
	DeviceTypeHumidifier    = "devices.types.humidifier"
	DeviceTypeDehumidifier  = "devices.types.dehumidifier"
	DeviceTypeIceMaker      = "devices.types.ice_maker"
	DeviceTypeAromaDiffuser = "devices.types.aroma_diffuser"
	DeviceTypeBox           = "devices.types.box"
)

type GoveeService struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

type CapabilityParameter struct {
	Unit     string `json:"unit,omitempty"`
	DataType string `json:"dataType"`
	Options  []struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	} `json:"options,omitempty"`
	Range *struct {
		Min       int `json:"min"`
		Max       int `json:"max"`
		Precision int `json:"precision"`
	} `json:"range,omitempty"`
	Fields []struct {
		FieldName    string `json:"fieldName"`
		DataType     string `json:"dataType"`
		Required     bool   `json:"required"`
		Size         *struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"size,omitempty"`
		ElementRange *struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"elementRange,omitempty"`
		ElementType string `json:"elementType,omitempty"`
		Options     []struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		} `json:"options,omitempty"`
		Range *struct {
			Min       int `json:"min"`
			Max       int `json:"max"`
			Precision int `json:"precision"`
		} `json:"range,omitempty"`
		Unit string `json:"unit,omitempty"`
	} `json:"fields,omitempty"`
}

type Capability struct {
	Type       string              `json:"type"`
	Instance   string              `json:"instance"`
	Parameters CapabilityParameter `json:"parameters"`
}

type Device struct {
	SKU          string       `json:"sku"`
	Device       string       `json:"device"`
	DeviceName   string       `json:"deviceName"`
	Type         string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
}

type DeviceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []Device `json:"data"`
}

type ControlRequest struct {
	RequestID string         `json:"requestId"`
	Payload   ControlPayload `json:"payload"`
}

type ControlPayload struct {
	SKU        string            `json:"sku"`
	Device     string            `json:"device"`
	Capability ControlCapability `json:"capability"`
}

type ControlCapability struct {
	Type     string      `json:"type"`
	Instance string      `json:"instance"`
	Value    interface{} `json:"value"`
}

type ControlResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewGoveeService(apiKey string) *GoveeService {
	return &GoveeService{
		client:  &http.Client{},
		apiKey:  apiKey,
		baseURL: "https://openapi.api.govee.com",
	}
}

func (s *GoveeService) GetDevices(ctx context.Context) ([]Device, error) {
	url := s.baseURL + "/router/api/v1/user/devices"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to %s: %w", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Govee-API-Key", s.apiKey)

	log.Printf("Making request to Govee API: %s", url)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request to Govee API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Govee API error response: %s", string(body))
		return nil, fmt.Errorf("govee api returned status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		log.Printf("Failed to parse response: %s", string(body))
		return nil, fmt.Errorf("parsing response body: %w", err)
	}

	if deviceResp.Code != 200 {
		return nil, fmt.Errorf("govee api error: %s (code: %d)", deviceResp.Message, deviceResp.Code)
	}

	log.Printf("Successfully fetched %d devices", len(deviceResp.Data))
	return deviceResp.Data, nil
}

func (s *GoveeService) ControlDevice(ctx context.Context, sku string, deviceID string, capability ControlCapability) error {
	url := s.baseURL + "/router/api/v1/device/control"

	// Validate capability
	if capability.Type == "" || capability.Instance == "" {
		return fmt.Errorf("invalid capability: type and instance are required")
	}

	request := ControlRequest{
		RequestID: uuid.New().String(),
		Payload: ControlPayload{
			SKU:        sku,
			Device:     deviceID,
			Capability: capability,
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	// Log the request payload for debugging
	log.Printf("Control request payload: %s", string(body))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request to %s: %w", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Govee-API-Key", s.apiKey)

	log.Printf("Making control request to Govee API: %s", url)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("making request to Govee API: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Govee API error response: %s", string(responseBody))
		return fmt.Errorf("govee api returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	var controlResp ControlResponse
	if err := json.Unmarshal(responseBody, &controlResp); err != nil {
		log.Printf("Failed to parse response: %s", string(responseBody))
		return fmt.Errorf("parsing response body: %w", err)
	}

	if controlResp.Code != 200 {
		log.Printf("Full response body: %s", string(responseBody))
		return fmt.Errorf("govee api error: %s (code: %d)", controlResp.Message, controlResp.Code)
	}

	log.Printf("Successfully controlled device %s", deviceID)
	return nil
}

	