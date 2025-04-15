package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/CyberRoute/graphspecter/pkg/cli"
	"github.com/CyberRoute/graphspecter/pkg/cmd"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/network"
	"github.com/CyberRoute/graphspecter/pkg/subscription"
)

func main() {

	// Parse all command-line flags.
	cfg := cmd.ParseFlags()

	// If subscribe flag is set, wait for user input before subscribing.
	if cfg.Subscribe {
		var query string
		if cfg.SubQuery != "" {
			query = cfg.SubQuery
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
		conn, err := subscription.SubscribeToQuery(cfg.WSURL, query)
		if err != nil {
			logger.Error("Subscription error: %v", err)
			os.Exit(1)
		}
		logger.Info("Subscription established. Listening for updates...")
		subscription.Listen(conn)
		os.Exit(0)
	}

	// If neither a schema file nor a base URL is provided, show usage and exit.
	if cfg.SchemaFile == "" && cfg.BaseURL == "" {
		flag.Usage()
		os.Exit(0)
	}

	// Configure logging.
	logger.SetupLogging(cfg.LogLevel, cfg.LogFile, !cfg.NoColor)

	// Handle schema parsing if the file option is provided.
	if cfg.SchemaFile != "" {
		cli.HandleSchemaFile(cfg.SchemaFile, cfg.List, cfg.Query, cfg.Mutation, cfg.AllQueries, cfg.AllMutations, cfg.MaxDepth)
		os.Exit(0)
	}

	cli.DisplayLogo()
	logger.Info("GraphSpecter v1.0.0 starting...")
	logger.Debug("Timeout set to %s", cfg.Timeout)

	// Create a context with the user-specified timeout.
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer timeoutCancel()

	// Set up target URLs for network operations.
	var targetURLs []string

	if cfg.Detect {
		// Detection mode.
		logger.Info("Detection mode enabled. Scanning for GraphQL endpoints...")
		detectedEndpoints, err := network.DetectAllGraphQLEndpointsWithContext(timeoutCtx, cfg.BaseURL, false)
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
		targetURLs = append(targetURLs, cfg.BaseURL)
		logger.Info("Using base URL as target: %s", cfg.BaseURL)
	}

	logger.Info("Starting GraphQL security audit...")

	// Common headers for all requests.
	headers := map[string]string{"Content-Type": "application/json"}

	// Add Authorization header if AUTH_TOKEN environment variable is set.
	if authToken := os.Getenv("AUTH_TOKEN"); authToken != "" {
		logger.Debug("Using authentication token from environment")
		headers["Authorization"] = "Bearer " + authToken
	}

	cli.AuditEndpoints(timeoutCtx, targetURLs, headers, cfg.OutputFile)
}
