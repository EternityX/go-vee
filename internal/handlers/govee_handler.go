package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/EternityX/go-vee/internal/service"
)

type GoveeHandler struct {
	service *service.GoveeService
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"description,omitempty"`
	Code        int    `json:"code"`
}

func NewGoveeHandler(service *service.GoveeService) *GoveeHandler {
	return &GoveeHandler{
		service: service,
	}
}

func sendErrorResponse(w http.ResponseWriter, message string, code int, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:       message,
		Description: description,
		Code:        code,
	})
}

func (h *GoveeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse webhook payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Error decoding webhook payload: %v", err)
		sendErrorResponse(w, "Bad request", http.StatusBadRequest, "Invalid request body")
		return
	}

	// Log the webhook payload for now
	log.Printf("Received webhook: %+v", payload)
}

func (h *GoveeHandler) HandleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "Only GET method is allowed for this endpoint")
		return
	}
	
	devices, err := h.service.GetDevices(r.Context())
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		description := "Failed to fetch devices from Govee API"
		if strings.Contains(err.Error(), "Govee API returned status") {
			description = err.Error()
		}
		sendErrorResponse(w, "Internal server error", http.StatusInternalServerError, description)
		return
	}

	response := struct {
		Success bool             `json:"success"`
		Data    []service.Device `json:"data"`
	}{
		Success: true,
		Data:    devices,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		sendErrorResponse(w, "Internal server error", http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

func (h *GoveeHandler) HandleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "Only POST method is allowed for this endpoint")
		return
	}

	// Parse the request body
	var controlRequest struct {
		SKU        string                    `json:"sku"`
		Device     string                    `json:"device"`
		Capability service.ControlCapability `json:"capability"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		sendErrorResponse(w, "Bad request", http.StatusBadRequest, "Failed to read request body")
		return
	}
	
	log.Printf("Received control request: %s", string(body))

	if err := json.Unmarshal(body, &controlRequest); err != nil {
		log.Printf("Error decoding control request: %v", err)
		sendErrorResponse(w, "Bad request", http.StatusBadRequest, "Invalid request body format")
		return
	}

	if controlRequest.SKU == "" || controlRequest.Device == "" {
		sendErrorResponse(w, "Bad request", http.StatusBadRequest, "Missing required fields: sku and device")
		return
	}

	if controlRequest.Capability.Type == "" || controlRequest.Capability.Instance == "" {
		sendErrorResponse(w, "Bad request", http.StatusBadRequest, "Missing required capability fields: type and instance")
		return
	}

	// Call the service to control the device
	err = h.service.ControlDevice(r.Context(), controlRequest.SKU, controlRequest.Device, controlRequest.Capability)
	if err != nil {
		log.Printf("Error controlling device: %v", err)
		description := "Failed to control device"
		if strings.Contains(err.Error(), "govee api") {
			description = err.Error()
		}
		sendErrorResponse(w, "Internal server error", http.StatusInternalServerError, description)
		return
	}

	response := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: "Device control command sent successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		sendErrorResponse(w, "Internal server error", http.StatusInternalServerError, "Failed to encode response")
		return
	}
}
