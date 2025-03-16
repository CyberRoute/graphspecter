package fingerprint

import (
	"fmt"
	"strings"
	"github.com/yourusername/black_hat_gql/pkg/network"
)

// DetectEngine sends several engine-specific queries and returns the detected engine name.
func DetectEngine(url string, headers map[string]string) (string, error) {
	if DetectApollo(url, headers) {
		return "Apollo", nil
	}
	// Additional engine detection functions can be added here.
	// For example:
	// if detectHasura(url, headers) { return "Hasura", nil }
	// if detectGraphQLPHP(url, headers) { return "GraphQL PHP", nil }
	return "", fmt.Errorf("no known GraphQL engine detected")
}

// DetectApollo sends a query with the @skip directive and checks the error messages
// to determine if the Apollo engine is in use.
func DetectApollo(url string, headers map[string]string) bool {
	const expectedErrorSubstring = `Directive "@skip" argument "if" of type "Boolean!" is required`

	result, err := network.SendGraphQLRequest(url, `query @skip { __typename }`, nil, headers)
	if err != nil {
		return false
	}

	// Check for the expected error message
	errors, ok := result["errors"].([]interface{})
	if !ok {
		return false
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
			return true
		}
	}

	return false
}