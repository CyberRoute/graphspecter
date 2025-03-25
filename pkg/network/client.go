package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/types"
)

// CommonPaths contains a list of potential GraphQL endpoints.
var CommonPaths = []string{
	"/",
	"/graphql",
	"/graphiql",
	"/v1/graphql",
	"/v2/graphql",
	"/v3/graphql",
	"/api/graphql",
	"/console",
	"/playground",
	"/gql",
	"/query",
	"/api",
	"/graphql/v1",
	"/graphql/v2",
	"/api/v1/graphql",
	"/api/v2/graphql",
	"/graph",
	"/graphql-api",
	"/graphql/console",
	"/graphql/playground",
	"/service-name/graphql", // Often used for microservices
	"/hasura/v1/graphql",    // Hasura specific
	"/altair",               // Altair GraphQL client
	"/explorer",             // GraphQL Explorer
}

// DefaultTimeout is the default timeout for HTTP requests.
const DefaultTimeout = 10 * time.Second

// SendGraphQLRequest sends a GraphQL request to the given endpoint.
// This is a backward compatibility wrapper for the context-aware version.
func SendGraphQLRequest(url string, query string, variables map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return SendGraphQLRequestWithContext(ctx, url, query, variables, headers)
}

// SendGraphQLRequestWithContext sends a GraphQL request to the given endpoint with context support.
func SendGraphQLRequestWithContext(ctx context.Context, url string, query string, variables map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	reqBody := types.GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: DefaultTimeout,
	}

	logger.Debug("Sending GraphQL request to %s", url)
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			logger.Debug("Request to %s was canceled", url)
			return nil, fmt.Errorf("request canceled by user or another operation completed first")
		} else if ctx.Err() == context.DeadlineExceeded {
			logger.Debug("Request to %s timed out", url)
			return nil, fmt.Errorf("request timed out, consider increasing timeout")
		} else {
			logger.Error("Error sending request: %v", err)
			return nil, fmt.Errorf("error sending request: %w", err)
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading response: %v", err)
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check content type to make sure we're getting JSON
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !containsSubstring(contentType, "json") {
		logger.Debug("Non-JSON response detected (Content-Type: %s)", contentType)
		return nil, fmt.Errorf("non-JSON response received (Content-Type: %s)", contentType)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If content starts with "<", it's likely HTML
		if len(body) > 0 && body[0] == '<' {
			logger.Debug("HTML response detected instead of JSON")
			return nil, fmt.Errorf("HTML response received instead of expected JSON")
		}
		logger.Error("Error parsing response: %v", err)
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	logger.Debug("Received response from %s, status: %d", url, resp.StatusCode)
	return result, nil
}

// DetectGraphQLEndpoint scans common endpoints appended to the base URL.
// This is a backward compatibility wrapper for the context-aware version.
func DetectGraphQLEndpoint(baseURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return DetectGraphQLEndpointWithContext(ctx, baseURL)
}

// DetectGraphQLEndpointWithContext scans common endpoints appended to the base URL with context support.
func DetectGraphQLEndpointWithContext(ctx context.Context, baseURL string) (string, error) {
	results, err := DetectAllGraphQLEndpointsWithContext(ctx, baseURL, true)
	if err != nil {
		return "", err
	}

	if len(results) > 0 {
		return results[0], nil
	}

	return "", fmt.Errorf("GraphQL endpoint not detected on any common paths")
}

// DetectAllGraphQLEndpointsWithContext scans and returns all valid GraphQL endpoints
// If stopOnFirst is true, it will stop after finding the first valid endpoint
func DetectAllGraphQLEndpointsWithContext(ctx context.Context, baseURL string, stopOnFirst bool) ([]string, error) {
	logger.Info("Starting endpoint detection for %s", baseURL)

	// Use concurrency for faster scanning
	var wg sync.WaitGroup
	resultChan := make(chan string, len(CommonPaths))

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Count of endpoints checked
	checkedEndpoints := 0
	var mutex sync.Mutex

	// Normalize base URL to ensure it doesn't end with a slash
	baseURL = strings.TrimRight(baseURL, "/")

	// Start concurrent checks for each potential endpoint
	for _, path := range CommonPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return // Context was cancelled (timeout or stopOnFirst)
			default:
				endpoint := baseURL + p
				logger.Debug("Checking endpoint: %s", endpoint)
				isValid, err := IsGraphQLEndpointWithContext(ctx, endpoint)

				mutex.Lock()
				checkedEndpoints++
				mutex.Unlock()

				if err != nil {
					logger.Debug("Error checking %s: %v", endpoint, err)
					return
				}

				if isValid {
					logger.Info("Found GraphQL endpoint at: %s", endpoint)
					resultChan <- endpoint

					// If stopOnFirst is true, cancel other goroutines
					if stopOnFirst {
						cancel()
					}
				}
			}
		}(path)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect all results
	var results []string

	// Keep collecting until channel is closed or context is cancelled
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// Channel closed, all goroutines finished
				goto DONE
			}
			if result != "" {
				results = append(results, result)
				// If stopOnFirst is true and we got a result, we can stop collecting
				if stopOnFirst {
					goto DONE
				}
			}
		case <-ctx.Done():
			// Context was cancelled
			goto DONE
		}
	}

DONE:
	if checkedEndpoints == 0 {
		return nil, fmt.Errorf("unable to check any GraphQL endpoints, possible network or server issue")
	}

	return results, nil
}

// IsGraphQLEndpoint sends a simple query to see if the response looks like GraphQL.
// This is a backward compatibility wrapper for the context-aware version.
func IsGraphQLEndpoint(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	isEndpoint, _ := IsGraphQLEndpointWithContext(ctx, url)
	return isEndpoint
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsSubstring(s, substring string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substring))
}

// GetFriendlyErrorMessage converts technical errors to user-friendly messages
func GetFriendlyErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Handle context cancellation errors
	if strings.Contains(errMsg, "context canceled") {
		return "Request was canceled - either by user interruption or another endpoint was found"
	}

	// Handle timeout errors
	if strings.Contains(errMsg, "context deadline exceeded") || strings.Contains(errMsg, "timeout") {
		return "Request timed out - consider increasing timeout with -timeout flag"
	}

	// Handle connection errors
	if strings.Contains(errMsg, "connection refused") {
		return "Connection refused - server may be down or not accepting connections"
	}

	// Return original error message if no friendly version is available
	return errMsg
}

// IsGraphQLEndpointWithContext sends a simple query to see if the response looks like GraphQL with context support.
func IsGraphQLEndpointWithContext(ctx context.Context, url string) (bool, error) {
	query := `query { __typename }`
	result, err := SendGraphQLRequestWithContext(ctx, url, query, nil, nil)
	if err != nil {
		// If we got HTML or non-JSON response, treat this as "not a GraphQL endpoint"
		// rather than a hard error
		if strings.Contains(err.Error(), "HTML response") ||
			strings.Contains(err.Error(), "non-JSON response") {
			logger.Debug("Endpoint %s is not a GraphQL endpoint: %v", url, err)
			return false, nil
		}
		// Context cancellation and timeouts are normal during parallel endpoint detection
		if strings.Contains(err.Error(), "canceled") ||
			strings.Contains(err.Error(), "timed out") {
			logger.Debug("Check for endpoint %s was interrupted: %v", url, err)
			return false, nil
		}
		return false, err
	}

	// Check for __typename in data or a non-empty errors array.
	if data, ok := result["data"].(map[string]interface{}); ok {
		if typename, exists := data["__typename"].(string); exists {
			if typename == "Query" || typename == "QueryRoot" || typename == "query_root" {
				return true, nil
			}
		}
	}
	if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
		return true, nil
	}
	return false, nil
}
