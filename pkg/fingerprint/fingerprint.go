package fingerprint

import (
	"context"
	"fmt"
	"strings"
	"github.com/CyberRoute/graphspecter/pkg/network"
	"github.com/CyberRoute/graphspecter/pkg/logger"
)

// DetectEngine sends several engine-specific queries and returns the detected engine name.
// This is a backward compatibility wrapper for the context-aware version.
func DetectEngine(url string, headers map[string]string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), network.DefaultTimeout)
	defer cancel()
	return DetectEngineWithContext(ctx, url, headers)
}

// DetectEngineWithContext sends several engine-specific queries and returns the detected engine name with context support.
func DetectEngineWithContext(ctx context.Context, url string, headers map[string]string) (string, error) {
	logger.Info("Starting GraphQL engine fingerprinting for %s", url)
	
	// Use a cancellable context for all detection routines
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	// Result channel to collect detection results
	resultChan := make(chan string, 4) // Buffered channel to avoid goroutine blocking
	
	// Start Apollo detection
	go func() {
		isApollo, err := DetectApolloWithContext(ctx, url, headers)
		if err != nil {
			logger.Debug("Apollo detection error: %v", err)
			return
		}
		if isApollo {
			logger.Info("Detected Apollo GraphQL engine")
			resultChan <- "Apollo"
			cancel() // Cancel other detections
		}
	}()
	
	// Additional engine detection functions can be added here as goroutines
	// For example:
	// go func() { 
	//     isHasura, _ := detectHasuraWithContext(ctx, url, headers)
	//     if isHasura {
	//         resultChan <- "Hasura"
	//         cancel()
	//     }
	// }()
	
	// Wait for a result or context cancellation
	select {
	case result := <-resultChan:
		return result, nil
	case <-ctx.Done():
		// Check if cancellation was due to a detection
		select {
		case result := <-resultChan:
			return result, nil
		default:
			// No detection happened, timeout or external cancellation
			return "", fmt.Errorf("no known GraphQL engine detected or detection timed out")
		}
	}
}

// DetectApollo sends a query with the @skip directive and checks the error messages
// to determine if the Apollo engine is in use.
// This is a backward compatibility wrapper for the context-aware version.
func DetectApollo(url string, headers map[string]string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), network.DefaultTimeout)
	defer cancel()
	isApollo, _ := DetectApolloWithContext(ctx, url, headers)
	return isApollo
}

// DetectApolloWithContext sends a query with the @skip directive and checks the error messages
// to determine if the Apollo engine is in use with context support.
func DetectApolloWithContext(ctx context.Context, url string, headers map[string]string) (bool, error) {
	const expectedErrorSubstring = `Directive "@skip" argument "if" of type "Boolean!" is required`

	logger.Debug("Testing for Apollo GraphQL engine at %s", url)
	result, err := network.SendGraphQLRequestWithContext(ctx, url, `query @skip { __typename }`, nil, headers)
	if err != nil {
		return false, err
	}

	// Check for the expected error message
	errors, ok := result["errors"].([]interface{})
	if !ok {
		return false, nil
	}

	for _, err := range errors {
		errMap, ok := err.(map[string]interface{})
		if !ok {
			continue
		}

		message, ok := errMap["message"].(string)
		if !ok {
			continue
		}

		if strings.Contains(message, expectedErrorSubstring) {
			return true, nil
		}
	}

	return false, nil
}