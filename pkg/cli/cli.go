// pkg/cli.go
package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/CyberRoute/graphspecter/pkg/introspection"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/schema"
	"github.com/CyberRoute/graphspecter/pkg/types"
)

func DisplayLogo() {
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

// SetupSignalHandler creates a cancellable context and registers a signal handler for graceful shutdown.
// It returns the new context and its cancel function.
func SetupSignalHandler(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
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

	return ctx, cancel
}

// HandleSchemaFile processes an introspection JSON file and handles schema-related operations.
func HandleSchemaFile(filePath, listOption, queryOption, mutationOption string, allQueries, allMutations bool, maxDepth int) {
	// Load the schema from file
	schemaObj, err := schema.LoadFromFile(filePath)
	if err != nil {
		logger.Error("Failed to load schema: %v", err)
		os.Exit(1)
	}

	// Handle the list option to print available queries and mutations
	if listOption != "" {
		PrintAvailableOperations(schemaObj, listOption)
		return
	}

	// Determine whether to print specific queries/mutations or all.
	if queryOption != "" || mutationOption != "" {
		allQueries = false
		allMutations = false
	} else if !allQueries && !allMutations {
		allQueries = true
		allMutations = true
	}

	// Print queries
	if allQueries || queryOption != "" {
		var queryNames []string
		if allQueries {
			queryNames = schema.ListQueries(schemaObj)
		} else {
			queryNames = strings.Split(queryOption, ",")
		}
		GenerateAndPrintOperations(schema.GenerateQuery, schemaObj, queryNames, maxDepth, "query")
	}

	// Print mutations
	if (allMutations || mutationOption != "") && schemaObj.Mutation != nil {
		var mutationNames []string
		if allMutations {
			mutationNames = schema.ListMutations(schemaObj)
		} else {
			mutationNames = strings.Split(mutationOption, ",")
		}
		GenerateAndPrintOperations(schema.GenerateMutation, schemaObj, mutationNames, maxDepth, "mutation")
	}
}

func GenerateAndPrintOperations(
	generateFn func(*types.GQLSchema, string, int) (string, error),
	schemaObj *types.GQLSchema,
	names []string,
	maxDepth int,
	opType string,
) {
	for _, name := range names {
		op, err := generateFn(schemaObj, name, maxDepth)
		if err != nil {
			logger.Error("Failed to generate %s for %s: %v", opType, name, err)
			continue
		}
		fmt.Println(op)
	}
}

// PrintAvailableOperations prints the names of queries and/or mutations in the schema.
func PrintAvailableOperations(schemaObj *types.GQLSchema, listOption string) {
	if listOption == "queries" || listOption == "all" {
		for _, queryName := range schema.ListQueries(schemaObj) {
			fmt.Printf("query %s\n", queryName)
		}
	}

	if (listOption == "mutations" || listOption == "all") && schemaObj.Mutation != nil {
		for _, mutationName := range schema.ListMutations(schemaObj) {
			fmt.Printf("mutation %s\n", mutationName)
		}
	}
}

// AuditEndpoints checks each target URL for introspection and writes the results to file.
func AuditEndpoints(timeoutCtx context.Context, targetURLs []string, headers map[string]string, outputFile string) {
	// Track if we found at least one endpoint with introspection enabled.
	introspectionEnabled := false
	var lastIntrospectionResult map[string]interface{}

	// Loop through each target URL.
	for _, targetURL := range targetURLs {
		logger.Info("Checking target: %s", targetURL)
		logger.Info("Checking if introspection is enabled on %s...", targetURL)
		introspectionResult, err := introspection.CheckIntrospectionWithContext(timeoutCtx, targetURL, headers)
		if err != nil {
			if strings.Contains(err.Error(), "HTML response") || strings.Contains(err.Error(), "non-JSON response") {
				logger.Warn("The endpoint %s doesn't appear to be a valid GraphQL endpoint: %v", targetURL, err)
				logger.Info("This may be a false positive or the endpoint requires special headers/authentication")
				continue
			}
			logger.Error("Error checking introspection on %s: %v", targetURL, err)
			continue
		}

		lastIntrospectionResult = introspectionResult

		if introspection.IsIntrospectionEnabled(introspectionResult) {
			logger.Warn("WARNING: Introspection is ENABLED on %s!", targetURL)
			outName := outputFile
			outName = generateOutputFileName(outputFile, targetURL)
			err = introspection.WriteIntrospectionToFile(introspectionResult, outName)
			if err != nil {
				logger.Error("Error writing introspection result to file: %v", err)
				continue
			}
			logger.Info("Introspection data saved to %s", outName)
			introspectionEnabled = true
		} else {
			logger.Info("Introspection appears to be disabled on %s", targetURL)
		}
	}

	// Output summary.
	if introspectionEnabled {
		logger.Warn("WARNING: Introspection is ENABLED on at least one endpoint!")
	} else if lastIntrospectionResult != nil {
		logger.Info("Introspection appears to be disabled on all checked endpoints")
	}
	logger.Info("Audit completed")
}

func generateOutputFileName(defaultFile, targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return defaultFile
	}
	cleanPath := strings.Trim(parsed.Path, "/")
	suffix := "root"
	if cleanPath != "" {
		segments := strings.Split(cleanPath, "/")
		suffix = segments[len(segments)-1]
	}
	baseName := strings.TrimSuffix(defaultFile, ".json")
	return fmt.Sprintf("%s_%s.json", baseName, suffix)
}
