package introspection

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/network"
	"os"
)

// IntrospectionQuery contains the full introspection query.
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
      }
    }
  }
}
`

// CheckIntrospection sends the introspection query to the target URL.
// This is a backward compatibility wrapper for the context-aware version.
func CheckIntrospection(url string, headers map[string]string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), network.DefaultTimeout)
	defer cancel()
	return CheckIntrospectionWithContext(ctx, url, headers)
}

// CheckIntrospectionWithContext sends the introspection query to the target URL with context support.
func CheckIntrospectionWithContext(ctx context.Context, url string, headers map[string]string) (map[string]interface{}, error) {
	logger.Info("Checking introspection at %s", url)
	result, err := network.SendGraphQLRequestWithContext(ctx, url, IntrospectionQuery, nil, headers)
	if err != nil {
		// Check for common errors and provide more user-friendly messages
		if ctx.Err() == context.Canceled {
			logger.Error("Introspection query was canceled")
			return nil, fmt.Errorf("operation canceled - either by user interruption or another operation completed first")
		} else if ctx.Err() == context.DeadlineExceeded {
			logger.Error("Introspection query timed out")
			return nil, fmt.Errorf("request timed out - try increasing timeout with the -timeout flag")
		}

		logger.Error("Introspection query failed: %v", err)
		return nil, err
	}
	logger.Debug("Received introspection response")
	return result, nil
}

// IsIntrospectionEnabled checks if introspection is enabled based on the response.
func IsIntrospectionEnabled(response map[string]interface{}) bool {
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return false
	}
	schema, ok := data["__schema"].(map[string]interface{})
	if !ok {
		return false
	}
	types, ok := schema["types"].([]interface{})
	return ok && len(types) > 0
}

// WriteIntrospectionToFile writes the introspection result to a file.
func WriteIntrospectionToFile(data map[string]interface{}, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	return nil
}
