package introspection

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/yourusername/black_hat_gql/pkg/network"
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
func CheckIntrospection(url string, headers map[string]string) (map[string]interface{}, error) {
	return network.SendGraphQLRequest(url, IntrospectionQuery, nil, headers)
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