package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	amaasclient "github.com/trendmicro/tm-v1-fs-golang-sdk"
)

// ScanResponse represents the response we'll send back to the Node.js application
type ScanResponse struct {
	IsSafe     bool     `json:"isSafe"`
	Message    string   `json:"message"`
	ScanID     string   `json:"scanId,omitempty"`
	Detections string   `json:"detections,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string   `json:"status"`
	Timestamp   string   `json:"timestamp"`
	CustomTags  []string `json:"customTags"`
	APIEndpoint string   `json:"apiEndpoint"`
}

// Get environment variable with default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Get custom tags from environment
func getCustomTags() []string {
	customTags := os.Getenv("FSS_CUSTOM_TAGS")
	if customTags == "" {
		return []string{}
	}
	return strings.Split(customTags, ",")
}

func main() {
	// Get configuration from environment variables
	apiKey := os.Getenv("FSS_API_KEY")
	region := getEnv("FSS_REGION", "us-1")
	externalAddr := os.Getenv("SCANNER_EXTERNAL_ADDR")
	useTLS := os.Getenv("SCANNER_USE_TLS") == "true"

	// Get custom tags
	customTags := getCustomTags()

	// Configure logging
	f, err := os.OpenFile("/app/scanner.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Initialize S3 logger
	initS3Logger()

	// Log startup configuration
	log.Printf("Scanner Service Starting")
	log.Printf("Configuration:")

	// Create AMaaS client - both modes use the SDK client interface
	var client *amaasclient.AmaasClient
	var endpoint string

	if externalAddr != "" {
		// External gRPC scanner mode
		log.Printf("- Mode: External Scanner (gRPC)")
		log.Printf("- Scanner Address: %s", externalAddr)
		log.Printf("- TLS: %v", useTLS)
		log.Printf("- Custom Tags: %v", customTags)
		endpoint = externalAddr

		var err error
		client, err = amaasclient.NewClientInternal("", externalAddr, useTLS, "")
		if err != nil {
			log.Fatalf("Failed to create external scanner client: %v", err)
		}
	} else {
		// SaaS SDK mode (default)
		if apiKey == "" {
			log.Fatal("FSS_API_KEY must be set when not using external scanner")
		}
		log.Printf("- Mode: SaaS SDK Scanner")
		log.Printf("- Region: %s", region)
		log.Printf("- Custom Tags: %v", customTags)
		endpoint = region

		var err error
		client, err = amaasclient.NewClient(apiKey, region)
		if err != nil {
			log.Fatalf("Failed to create SaaS SDK scanner client: %v", err)
		}
	}

	startHTTPServer(client, customTags, endpoint)
}

// startHTTPServer starts the HTTP server with the given client
func startHTTPServer(client *amaasclient.AmaasClient, customTags []string, endpoint string) {

	// Enable digest calculation to get file hashes (SHA1, SHA256) for audit purposes
	// Note: Digest is disabled by default. We enable it for security auditing.
	// Only disable if using AmaasReader with remote files to reduce network traffic.

	// Handle scan requests
	http.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get headers
		filename := r.Header.Get("X-Filename")
		if filename == "" {
			filename = "unknown"
		}

		scanMethod := r.Header.Get("X-Scan-Method")
		if scanMethod == "" {
			scanMethod = "buffer" // default to buffer method
		}

		filePath := r.Header.Get("X-File-Path")

		// Get digest configuration
		digestEnabled := r.Header.Get("X-Digest-Enabled")
		if digestEnabled == "false" {
			client.SetDigestDisable()
			log.Printf("Digest calculation disabled for this scan")
		} else {
			// Digest is enabled by default, no action needed
			log.Printf("Digest calculation enabled for this scan")
		}

		// Get PML configuration
		pmlEnabled := r.Header.Get("X-PML-Enabled")
		if pmlEnabled == "true" {
			client.SetPMLEnable()
			log.Printf("PML (Predictive Machine Learning) detection enabled")
		}

		// Get SPN Feedback configuration
		spnFeedbackEnabled := r.Header.Get("X-SPN-Feedback-Enabled")
		if spnFeedbackEnabled == "true" {
			client.SetFeedbackEnable()
			log.Printf("SPN feedback enabled")
		}

		// Get Verbose configuration
		verboseEnabled := r.Header.Get("X-Verbose-Enabled")
		if verboseEnabled == "true" {
			client.SetVerboseEnable()
			log.Printf("Verbose scan result enabled")
		}

		// Get Active Content Detection configuration
		activeContentEnabled := r.Header.Get("X-Active-Content-Enabled")
		if activeContentEnabled == "true" {
			client.SetActiveContentEnable()
			log.Printf("Active content detection enabled (PDF scripts, Office macros)")
		}

		// Generate unique identifier
		identifier := time.Now().Format("20060102150405") + "-" + filepath.Base(filename)

		// Initial tags with key=value format
		tags := append([]string{
			"app=finguard",                           // Application tag
			"file_type=" + filepath.Ext(filename),    // File extension tag
			"scan_method=" + scanMethod,              // Scan method tag
			"ml_enabled=" + pmlEnabled,               // PML detection status
			"spn_feedback=" + spnFeedbackEnabled,     // SPN feedback status
			"active_content=" + activeContentEnabled, // Active content detection status
		}, customTags...)

		var scanResult string
		var err error

		// Choose scan method based on header
		if scanMethod == "file" && filePath != "" {
			// Scan using file method
			log.Printf("Starting file scan for: %s with tags: %v", filePath, tags)
			log.Printf("SDK Call: client.ScanFile(filePath=%s, tags=%v)", filePath, tags)
			scanResult, err = client.ScanFile(filePath, tags)
			if err == nil {
				log.Printf("SDK Response: client.ScanFile() completed successfully")
			}
		} else {
			// Scan using buffer method (default)
			// Read file data
			data, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				log.Printf("Error reading request body: %v", readErr)
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			log.Printf("Starting buffer scan for file: %s with tags: %v", identifier, tags)
			log.Printf("SDK Call: client.ScanBuffer(data=[]byte[%d bytes], identifier=%s, tags=%v)", len(data), identifier, tags)
			scanResult, err = client.ScanBuffer(data, identifier, tags)
			if err == nil {
				log.Printf("SDK Response: client.ScanBuffer() completed successfully")
			}
		}

		if err != nil {
			log.Printf("Scan error for %s: %v", identifier, err)
			http.Error(w, "Scanning failed", http.StatusInternalServerError)
			return
		}

		// Parse scan result to extract malware names, file hashes, and determine if file is safe
		isSafe := true // Default to safe unless malware is found
		var scanData map[string]interface{}
		if err := json.Unmarshal([]byte(scanResult), &scanData); err == nil {
			// Extract file hashes for logging
			if fileSha1, ok := scanData["fileSha1"].(string); ok && fileSha1 != "" {
				log.Printf("File SHA1: %s", fileSha1)
			}
			if fileSha256, ok := scanData["fileSha256"].(string); ok && fileSha256 != "" {
				log.Printf("File SHA256: %s", fileSha256)
			}

			// Check if malware was found by examining the result.atse.malwareCount field
			if result, ok := scanData["result"].(map[string]interface{}); ok {
				if atse, ok := result["atse"].(map[string]interface{}); ok {
					if malwareCount, ok := atse["malwareCount"].(float64); ok && malwareCount > 0 {
						isSafe = false
						log.Printf("Malware detected! Malware count: %.0f", malwareCount)
					}

					// Extract malware names from the malware array
					if malwares, ok := atse["malware"].([]interface{}); ok {
						for _, malware := range malwares {
							if malwareMap, ok := malware.(map[string]interface{}); ok {
								if malwareName, ok := malwareMap["name"].(string); ok {
									tags = append(tags, "malware_name="+malwareName)
									log.Printf("Malware name: %s", malwareName)
								}
							}
						}
					}
				}
			}

			// Also check foundMalwares for backward compatibility
			if foundMalwares, ok := scanData["foundMalwares"].([]interface{}); ok && len(foundMalwares) > 0 {
				isSafe = false
				for _, malware := range foundMalwares {
					if malwareMap, ok := malware.(map[string]interface{}); ok {
						if malwareName, ok := malwareMap["malwareName"].(string); ok {
							tags = append(tags, "malware_name="+malwareName)
							log.Printf("Malware name (from foundMalwares): %s", malwareName)
						}
					}
				}
			}
		}

		// Prepare response based on scan result
		response := ScanResponse{
			IsSafe:     isSafe,
			Message:    scanResult,
			ScanID:     identifier,
			Tags:       tags,
			Detections: scanResult,
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}

		log.Printf("Scan completed for %s: %s with tags: %v", identifier, scanResult, tags)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := "healthy"

		// Verify scanner client is initialized
		if client == nil {
			status = "unhealthy"
		}

		response := HealthResponse{
			Status:      status,
			Timestamp:   time.Now().Format(time.RFC3339),
			CustomTags:  customTags,
			APIEndpoint: endpoint,
		}

		w.Header().Set("Content-Type", "application/json")
		if status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response)
	})

	// S3 object storage endpoints
	http.HandleFunc("/s3/buckets", handleListBuckets(client))
	http.HandleFunc("/s3/objects", handleListObjects(client))
	http.HandleFunc("/s3/scan", handleScanS3Object(client))

	// Start the server
	log.Printf("Scanner service starting on :3001")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
