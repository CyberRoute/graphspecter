package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yourusername/black_hat_gql/pkg/types"
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
}

// SendGraphQLRequest sends a GraphQL request to the given endpoint.
func SendGraphQLRequest(url string, query string, variables map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	reqBody := types.GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return result, nil
}

// DetectGraphQLEndpoint scans common endpoints appended to the base URL.
func DetectGraphQLEndpoint(baseURL string) (string, error) {
	for _, path := range CommonPaths {
		endpoint := baseURL + path
		fmt.Printf("[*] Checking endpoint: %s\n", endpoint)
		if IsGraphQLEndpoint(endpoint) {
			fmt.Printf("[+] Found GraphQL at: %s\n", endpoint)
			return endpoint, nil
		}
	}
	return "", fmt.Errorf("GraphQL endpoint not detected on any common paths")
}

// IsGraphQLEndpoint sends a simple query to see if the response looks like GraphQL.
func IsGraphQLEndpoint(url string) bool {
	query := `query { __typename }`
	result, err := SendGraphQLRequest(url, query, nil, nil)
	if err != nil {
		return false
	}

	// Check for __typename in data or a non-empty errors array.
	if data, ok := result["data"].(map[string]interface{}); ok {
		if typename, exists := data["__typename"].(string); exists {
			if typename == "Query" || typename == "QueryRoot" || typename == "query_root" {
				return true
			}
		}
	}
	if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
		return true
	}
	return false
}