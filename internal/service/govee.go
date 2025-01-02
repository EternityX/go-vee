package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/EternityX/go-vee/internal/service/lan"
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
	useLAN  bool
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
		FieldName string `json:"fieldName"`
		DataType  string `json:"dataType"`
		Required  bool   `json:"required"`
		Size      *struct {
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
	Code    int      `json:"code"`
	Message string   `json:"message"`
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

type RGBColor struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
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

func NewGoveeService(apiKey string, useLAN bool) *GoveeService {
	return &GoveeService{
		client:  &http.Client{},
		apiKey:  apiKey,
		baseURL: "https://openapi.api.govee.com",
		useLAN:  useLAN,
	}
}

// Fetches devices from the Govee cloud API
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

// Controls a device using either LAN or the Govee cloud API
func (s *GoveeService) ControlDevice(ctx context.Context, sku string, deviceID string, capability ControlCapability) error {
	if s.useLAN {
		devices, err := lan.DiscoverDevices(2 * time.Second)
		if err == nil {
			// Look for matching device
			for _, device := range devices {
				if device.Msg.Data.Device == deviceID {
					// Found device on LAN, try to control it
					var err error

					switch capability.Type {
					case "devices.capabilities.on_off":
						if val, ok := capability.Value.(float64); ok {
							if val == 1 {
								err = lan.TurnOn(device.Msg.Data.IP)
							} else {
								err = lan.TurnOff(device.Msg.Data.IP)
							}
						}
					case "devices.capabilities.range":
						if val, ok := capability.Value.(float64); ok {
							err = lan.SetBrightness(device.Msg.Data.IP, int(val))
						}
					case "devices.capabilities.color_setting":
						if colorInt, ok := capability.Value.(float64); ok {
							r := int((uint32(colorInt) >> 16) & 0xFF)
							g := int((uint32(colorInt) >> 8) & 0xFF)
							b := int(uint32(colorInt) & 0xFF)

							err = lan.SetColor(device.Msg.Data.IP, r, g, b)
						}
					}

					if err == nil {
						log.Printf("Successfully controlled device %s via LAN", deviceID)
						return nil
					}
					log.Printf("Failed to control device via LAN, falling back to cloud API: %v", err)
				}
			}
		} else {
			log.Printf("Failed to discover LAN devices, falling back to cloud API: %v", err)
		}
	}

	// Fall back to cloud API
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
