package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/CyberRoute/graphspecter/pkg/fingerprint"
	"github.com/CyberRoute/graphspecter/pkg/introspection"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/network"
)

func main() {
	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Setup graceful shutdown on interrupt
	setupSignalHandler(cancel)
	
	// Flags:
	// -base: Base URL (e.g. "http://192.168.1.1:5013")
	// -detect: Enable detection mode (scan common endpoints)
	// -fingerprint: Enable GraphQL engine fingerprinting
	// -output: File to save introspection results
	baseURL := flag.String("base", "", "Base URL of the target (e.g. http://192.168.1.1:5013)")
	detect := flag.Bool("detect", false, "Enable detection mode to find a GraphQL endpoint")
	fingerprintFlag := flag.Bool("fingerprint", false, "Enable GraphQL engine fingerprinting")
	outputFile := flag.String("output", "introspection.json", "Output file for introspection results")
	timeout := flag.Duration("timeout",1*time.Second, "Timeout for operations (e.g., 30s, 1m)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFile := flag.String("log-file", "", "Log to file in addition to stdout")
	noColor := flag.Bool("no-color", false, "Disable colored output")

	flag.Parse()
	
	// Configure logging
	setupLogging(*logLevel, *logFile, !*noColor)
	
	// Create a context with the user-specified timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, *timeout)
	defer timeoutCancel()
	
	if *baseURL == "" {
		flag.Usage()
		os.Exit(0)
	}

	// Log startup information
	displayLogo()
	logger.Info("GraphSpecter v1.0.0 starting...")
	logger.Debug("Timeout set to %s", *timeout)

	// Set up context for network operations
	var targetURLs []string
	
	if *detect {
		// Detection mode
		logger.Info("Detection mode enabled. Scanning for GraphQL endpoints...")
		// Check all endpoints by default
		detectedEndpoints, err := network.DetectAllGraphQLEndpointsWithContext(timeoutCtx, *baseURL, false)
		if err != nil {
			logger.Error("Detection failed: %v", err)
			os.Exit(1)
		}
		if len(detectedEndpoints) == 0 {
			logger.Error("No GraphQL endpoints detected")
			os.Exit(1)
		}
		targetURLs = detectedEndpoints
		logger.Info("Found %d GraphQL endpoints", len(targetURLs))

	} else {
		// Use the base URL directly if no endpoint/detection is provided
		targetURLs = append(targetURLs, *baseURL)
		logger.Info("Using base URL as target: %s", *baseURL)
	}

	logger.Info("Starting GraphQL security audit...")
	
	// Common headers for all requests
	headers := map[string]string{"Content-Type": "application/json"}
	
	// Add Authorization header if AUTH_TOKEN environment variable is set
	if authToken := os.Getenv("AUTH_TOKEN"); authToken != "" {
		logger.Debug("Using authentication token from environment")
		headers["Authorization"] = "Bearer " + authToken
	}

	// Track if we found at least one endpoint with introspection enabled
	introspectionEnabled := false
	var lastIntrospectionResult map[string]interface{}

	// If fingerprint flag is enabled, attempt to detect the GraphQL engine
	if *fingerprintFlag {
		// If detection mode is enabled, fingerprint each endpoint
		if *detect {
			for _, targetURL := range targetURLs {
				logger.Info("Fingerprinting GraphQL engine on %s...", targetURL)
				engine, err := fingerprint.DetectEngineWithContext(timeoutCtx, targetURL, headers)
				if err != nil {
					logger.Warn("Could not determine GraphQL engine for %s: %v", targetURL, err)
				} else {
					logger.Info("Discovered GraphQL Engine on %s: %s", targetURL, engine)
				}
			}
		} else {
			// Just fingerprint the base URL
			logger.Info("Fingerprinting GraphQL engine on %s...", *baseURL)
			engine, err := fingerprint.DetectEngineWithContext(timeoutCtx, *baseURL, headers)
			if err != nil {
				logger.Warn("Could not determine GraphQL engine for %s: %v", *baseURL, err)
			} else {
				logger.Info("Discovered GraphQL Engine on %s: %s", *baseURL, engine)
			}
		}
	}
	
	// Check each target URL
	if len(targetURLs) > 1 {
		for _, targetURL := range targetURLs {
			logger.Info("Checking target: %s", targetURL)
			
			// Check introspection
			logger.Info("Checking if introspection is enabled on %s...", targetURL)
			introspectionResult, err := introspection.CheckIntrospectionWithContext(timeoutCtx, targetURL, headers)
			if err != nil {
				// Check if it's a formatting error (HTML or non-JSON response)
				if strings.Contains(err.Error(), "HTML response") || 
				   strings.Contains(err.Error(), "non-JSON response") {
					logger.Warn("The endpoint %s doesn't appear to be a valid GraphQL endpoint: %v", targetURL, err)
					logger.Info("This may be a false positive or the endpoint requires special headers/authentication")
					continue
				}
				
				logger.Error("Error checking introspection on %s: %v", targetURL, err)
				continue
			}
	
			// Save the last valid introspection result for output
			lastIntrospectionResult = introspectionResult
			
			if introspection.IsIntrospectionEnabled(introspectionResult) {
				logger.Warn("WARNING: Introspection is ENABLED on %s!", targetURL)
				
				// Write introspection result to file
				outputName := *outputFile
				if len(targetURLs) > 1 {
					// For multiple endpoints, add endpoint to filename
					parts := strings.Split(targetURL, "/")
					endpointPart := "root"
					if len(parts) > 3 {
						endpointPart = strings.ReplaceAll(strings.Join(parts[3:], "_"), "/", "_")
						if endpointPart == "" {
							endpointPart = "root"
						}
					}
					ext := ".json"
					baseName := strings.TrimSuffix(*outputFile, ext)
					outputName = fmt.Sprintf("%s_%s%s", baseName, endpointPart, ext)
				}
				
				err = introspection.WriteIntrospectionToFile(introspectionResult, outputName)
				if err != nil {
					logger.Error("Error writing introspection result to file: %v", err)
					continue
				}
				
				logger.Info("Introspection data saved to %s", outputName)
				introspectionEnabled = true
				
				// Don't stop here - continue checking other endpoints even if this one has introspection enabled
			} else {
				logger.Info("Introspection appears to be disabled on %s", targetURL)
			}
		}

	}
	
	// Output summary
	if introspectionEnabled {
		logger.Warn("WARNING: Introspection is ENABLED on at least one endpoint!")
	} else if lastIntrospectionResult != nil {
		logger.Info("Introspection appears to be disabled on all checked endpoints")
	}
	logger.Info("Audit completed")
}

// setupSignalHandler registers signal handlers for graceful shutdown
func setupSignalHandler(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	
	go func() {
		<-c
		logger.Info("Received interrupt signal, shutting down...")
		cancel()
		// Give operations a chance to clean up
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}

func displayLogo() {
	logo := `
   ____                 _    ____                  _            
  / ___|_ __ __ _ _ __ | |__/ ___| _ __   ___  ___| |_ ___ _ __ 
 | |  _| '__/ _` + "`" + ` | '_ \| '_ \___ \| '_ \ / _ \/ __| __/ _ \ '__|
 | |_| | | | (_| | |_) | | | |__) | |_) |  __/ (__| ||  __/ |   
  \____|_|  \__,_| .__/|_| |_|____/| .__/ \___|\___|\__\___|_|   
                 |_|               |_|                           
`
	fmt.Println(logo)
}

// setupLogging configures the logger based on command line flags
func setupLogging(level string, logFilePath string, useColors bool) {
	// Set log level
	switch level {
	case "debug":
		logger.SetLevel(logger.LevelDebug)
	case "info":
		logger.SetLevel(logger.LevelInfo)
	case "warn":
		logger.SetLevel(logger.LevelWarn)
	case "error":
		logger.SetLevel(logger.LevelError)
	default:
		logger.SetLevel(logger.LevelInfo)
	}
	
	// Set up log file if specified
	if logFilePath != "" {
		err := logger.SetLogFile(logFilePath)
		if err != nil {
			fmt.Printf("Error setting up log file: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Configure color output
	logger.EnableColors(useColors)
}