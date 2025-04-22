package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/CyberRoute/graphspecter/pkg/cli"
	"github.com/CyberRoute/graphspecter/pkg/cmd"
	"github.com/CyberRoute/graphspecter/pkg/config"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/network"
	"github.com/CyberRoute/graphspecter/pkg/subscription"
)

func main() {
	// Parse all command-line flags.
	cfg := cmd.ParseFlags()

	if cfg.ConfigFile != "" {
		fileCfg, err := config.LoadConfigFile(cfg.ConfigFile)
		if err != nil {
			logger.Fatal("Error loading config file: %v", err)
		}
		config.ApplyFileConfigToCLIConfig(fileCfg, cfg)
	}
	// Batch execution mode: execute all .graphql files in a directory with vars
	if cfg.BatchDir != "" {
		if cfg.BaseURL == "" {
			logger.Fatal("--base is required for batch execution")
		}
		logger.Info("Batch mode: scanning directory %s", cfg.BatchDir)
		files, err := filepath.Glob(filepath.Join(cfg.BatchDir, "*.graphql"))
		if err != nil {
			logger.Fatal("Error scanning batch directory: %v", err)
		}
		// regex to find operation definitions
		opRegex := regexp.MustCompile(`(?m)^(?:query|mutation)\s+([A-Za-z0-9_]+)`)
		for _, qf := range files {
			contentBytes, err := os.ReadFile(qf)
			if err != nil {
				logger.Error("Skipping %s: %v", qf, err)
				continue
			}
			content := string(contentBytes)
			locs := opRegex.FindAllStringSubmatchIndex(content, -1)
			if len(locs) == 0 {
				logger.Error("No operations found in %s", qf)
				continue
			}

			// load variables file if present
			varsFile := strings.TrimSuffix(qf, ".graphql") + ".json"
			var vars map[string]interface{}
			if data, err := os.ReadFile(varsFile); err == nil {
				json.Unmarshal(data, &vars)
			}

			// prepare headers
			headers := map[string]string{"Content-Type": "application/json"}
			if auth := os.Getenv("AUTH_TOKEN"); auth != "" {
				headers["Authorization"] = "Bearer " + auth
			}

			// execute each operation separately
			for i, loc := range locs {
				start := loc[0]
				end := len(content)
				if i+1 < len(locs) {
					end = locs[i+1][0]
				}
				opDoc := content[start:end]
				// extract operation name
				hdr := opRegex.FindStringSubmatch(opDoc)
				opName := hdr[1]

				res, err := network.SendGraphQLRequestWithContext(context.Background(), cfg.BaseURL, opDoc, vars, headers)
				if err != nil {
					logger.Error("%s (in %s) failed: %v", opName, filepath.Base(qf), err)
					continue
				}
				out, _ := json.MarshalIndent(res, "", "  ")
				fmt.Printf("Result for %s (from %s):\n%s\n", opName, filepath.Base(qf), string(out))
			}
		}
		os.Exit(0)
	}

	// If execute flag is set, run provided query or mutation
	if cfg.Execute {
		if cfg.BaseURL == "" {
			logger.Fatal("--base is required when using --execute")
		}
		// Load query
		var query string
		if cfg.QueryString != "" {
			query = cfg.QueryString
		} else if cfg.QueryFile != "" {
			data, err := os.ReadFile(cfg.QueryFile)
			if err != nil {
				logger.Fatal("Error reading query file: %v", err)
			}
			query = string(data)
		} else {
			logger.Fatal("No query provided: use --query-string or --query-file")
		}

		// Parse variables
		var variables map[string]interface{}
		if cfg.Variables != "" {
			if err := json.Unmarshal([]byte(cfg.Variables), &variables); err != nil {
				logger.Fatal("Error parsing variables JSON: %v", err)
			}
		} else if cfg.VariablesFile != "" {
			data, err := os.ReadFile(cfg.VariablesFile)
			if err != nil {
				logger.Fatal("Error reading variables file: %v", err)
			}
			if err := json.Unmarshal(data, &variables); err != nil {
				logger.Fatal("Error parsing variables file JSON: %v", err)
			}
		}

		// Configure logging before request
		logger.SetupLogging(cfg.LogLevel, cfg.LogFile, !cfg.NoColor)

		// Prepare context
		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer timeoutCancel()

		// Prepare headers
		headers := map[string]string{"Content-Type": "application/json"}
		if cfg.Headers != nil {
			for k, v := range cfg.Headers {
				headers[k] = v
			}
		}
		if authToken := os.Getenv("AUTH_TOKEN"); authToken != "" {
			headers["Authorization"] = "Bearer " + authToken
		}

		// Execute request
		resp, err := network.SendGraphQLRequestWithContext(timeoutCtx, cfg.BaseURL, query, variables, headers)
		if err != nil {
			logger.Fatal("Execution error: %v", err)
		}

		// Pretty-print the JSON response
		output, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			logger.Fatal("Error formatting response: %v", err)
		}
		fmt.Println(string(output))
		os.Exit(0)
	}

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
	logger.Debug("→ Timeout set to %s", cfg.Timeout)

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
	if cfg.Headers != nil {
		for k, v := range cfg.Headers {
			headers[k] = v
		}
	}
	logger.Debug("→ Using headers: %+v", headers)
	// Add Authorization header if AUTH_TOKEN environment variable is set.
	if authToken := os.Getenv("AUTH_TOKEN"); authToken != "" {
		logger.Debug("→ Using authentication token from environment")
		headers["Authorization"] = "Bearer " + authToken
	}

	cli.AuditEndpoints(timeoutCtx, targetURLs, headers, cfg.OutputFile)
}
