package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/EternityX/go-vee/internal/handlers"
	"github.com/EternityX/go-vee/internal/service"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Govee-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	var apiKeyFlag string
	var portFlag string
	var lanFlag bool

	flag.StringVar(&apiKeyFlag, "api-key", "", "Govee API key")
	flag.StringVar(&portFlag, "port", "", "Port to listen on")
	flag.BoolVar(&lanFlag, "lan", true, "Enable LAN discovery (default: true)")
	flag.Parse()

	apiKey := apiKeyFlag
	if apiKey == "" {
		apiKey = os.Getenv("GOVEE_API_KEY")
	}

	// Only require API key if LAN mode is disabled
	if !lanFlag && apiKey == "" {
		log.Fatal("Govee API key is required when LAN mode is disabled. Provide it via -api-key flag or GOVEE_API_KEY environment variable")
	}

	goveeService := service.NewGoveeService(apiKey, lanFlag)
	goveeHandler := handlers.NewGoveeHandler(goveeService)

	mux := http.NewServeMux()

	// Handle devices endpoint
	mux.HandleFunc("/api/v1/devices", goveeHandler.HandleDevices)
	mux.HandleFunc("/api/v1/devices/control", goveeHandler.HandleControl)
	mux.HandleFunc("/api/v1/devices/lan", goveeHandler.HandleLANDevices)

	// Apply middleware
	handler := corsMiddleware(loggingMiddleware(mux))

	port := portFlag
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			log.Fatal("Port is required. Provide it via -port flag or PORT environment variable")
		}
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("LAN Discovery: %t", lanFlag)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
