package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CyberRoute/graphspecter/pkg/cli"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/network"
	"github.com/CyberRoute/graphspecter/pkg/subscription"
)

func main() {
	// Define flags.
	baseURL := flag.String("base", "", "Base URL of the target (e.g. http://192.168.1.1:5013)")
	detect := flag.Bool("detect", false, "Enable detection mode to find a GraphQL endpoint")
	outputFile := flag.String("output", "introspection.json", "Output file for introspection results")
	timeout := flag.Duration("timeout", 1*time.Second, "Timeout for operations (e.g., 30s, 1m)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFile := flag.String("log-file", "", "Log to file in addition to stdout")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	maxDepth := flag.Int("max-depth", 10, "Set the maximum level of nested field traversal when generating GraphQL selection sets from the introspection schema")
	// Schema parsing options.
	schemaFile := flag.String("schema-file", "", "File with the GraphQL schema (introspection JSON)")
	list := flag.String("list", "", "Parse GraphQL schema and list queries, mutations or both (valid values: 'queries', 'mutations' or 'all')")
	query := flag.String("query", "", "Only print named queries (comma-separated list of query names)")
	mutation := flag.String("mutation", "", "Only print named mutations (comma-separated list of mutation names)")
	allQueries := flag.Bool("all-queries", false, "Only print queries (by default both queries and mutations are printed)")
	allMutations := flag.Bool("all-mutations", false, "Only print mutations (by default both queries and mutations are printed)")
	subscribe := flag.Bool("subscribe", false, "Enable subscription mode to listen for paste updates")
	subQuery := flag.String("sub-query", "", "GraphQL subscription query to execute")
	wsURL := flag.String("ws-url", "ws://192.168.1.100:5013/subscriptions", "WebSocket URL for subscriptions")

	// Parse all command-line flags.
	flag.Parse()

	// If subscribe flag is set, wait for user input before subscribing.
	if *subscribe {
		var query string
		if *subQuery != "" {
			query = *subQuery
		} else {
			fmt.Println("Subscription mode enabled. Please enter your subscription query:")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				logger.Error("Error reading input: %v", err)
				os.Exit(1)
			}
			query = strings.TrimSpace(input)
		}

		// Attempt to subscribe using the generic function that tries both message types.
		conn, err := subscription.SubscribeToQuery(*wsURL, query)
		if err != nil {
			logger.Error("Subscription error: %v", err)
			os.Exit(1)
		}
		logger.Info("Subscription established. Listening for updates...")
		subscription.Listen(conn)
		os.Exit(0)
	}

	// If neither a schema file nor a base URL is provided, show usage and exit.
	if *schemaFile == "" && *baseURL == "" {
		flag.Usage()
		os.Exit(0)
	}

	// Configure logging.
	logger.SetupLogging(*logLevel, *logFile, !*noColor)

	// Handle schema parsing if the file option is provided.
	if *schemaFile != "" {
		cli.HandleSchemaFile(*schemaFile, *list, *query, *mutation, *allQueries, *allMutations, *maxDepth)
		os.Exit(0)
	}

	cli.DisplayLogo()
	logger.Info("GraphSpecter v1.0.0 starting...")
	logger.Debug("Timeout set to %s", *timeout)

	// Create a context with the user-specified timeout.
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), *timeout)
	defer timeoutCancel()

	// Set up target URLs for network operations.
	var targetURLs []string

	if *detect {
		// Detection mode.
		logger.Info("Detection mode enabled. Scanning for GraphQL endpoints...")
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
		// Use the base URL directly if no detection is provided.
		targetURLs = append(targetURLs, *baseURL)
		logger.Info("Using base URL as target: %s", *baseURL)
	}

	logger.Info("Starting GraphQL security audit...")

	// Common headers for all requests.
	headers := map[string]string{"Content-Type": "application/json"}

	// Add Authorization header if AUTH_TOKEN environment variable is set.
	if authToken := os.Getenv("AUTH_TOKEN"); authToken != "" {
		logger.Debug("Using authentication token from environment")
		headers["Authorization"] = "Bearer " + authToken
	}

	cli.AuditEndpoints(timeoutCtx, targetURLs, headers, *outputFile)
}
