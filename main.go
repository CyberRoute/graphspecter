package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/CyberRoute/graphspecter/pkg/cli"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/network"
)

func main() {
	// Create a cancellable context for graceful shutdown
	ctx, cancel := cli.SetupSignalHandler(context.Background())
	defer cancel()

	baseURL := flag.String("base", "", "Base URL of the target (e.g. http://192.168.1.1:5013)")
	detect := flag.Bool("detect", false, "Enable detection mode to find a GraphQL endpoint")
	outputFile := flag.String("output", "introspection.json", "Output file for introspection results")
	timeout := flag.Duration("timeout", 1*time.Second, "Timeout for operations (e.g., 30s, 1m)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFile := flag.String("log-file", "", "Log to file in addition to stdout")
	noColor := flag.Bool("no-color", false, "Disable colored output")

	// Schema parsing options
	schemaFile := flag.String("schema-file", "", "File with the GraphQL schema (introspection JSON)")
	listOption := flag.String("list", "", "Parse GraphQL schema and list queries, mutations or both (valid values: 'queries', 'mutations' or 'all')")
	queryOption := flag.String("query", "", "Only print named queries (comma-separated list of query names)")
	mutationOption := flag.String("mutation", "", "Only print named mutations (comma-separated list of mutation names)")
	allQueriesFlag := flag.Bool("all-queries", false, "Only print queries (by default both queries and mutations are printed)")
	allMutationsFlag := flag.Bool("all-mutations", false, "Only print mutations (by default both queries and mutations are printed)")

	flag.Parse()
	// If neither a schema file nor a base URL is provided, show usage documentation and exit.
	if *schemaFile == "" && *baseURL == "" {
		flag.Usage()
		os.Exit(0)
	}
	// Configure logging
	logger.SetupLogging(*logLevel, *logFile, !*noColor)

	// Handle schema parsing if the file option is provided
	if *schemaFile != "" {
		cli.HandleSchemaFile(*schemaFile, *listOption, *queryOption, *mutationOption, *allQueriesFlag, *allMutationsFlag)
		os.Exit(0)
	}
	cli.DisplayLogo()
	logger.Info("GraphSpecter v1.0.0 starting...")
	logger.Debug("Timeout set to %s", *timeout)

	// Create a context with the user-specified timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, *timeout)
	defer timeoutCancel()

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
	cli.AuditEndpoints(timeoutCtx, targetURLs, headers, *outputFile)
}
