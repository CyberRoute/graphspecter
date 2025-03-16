// pkg/generator/query_generator.go
package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const MAX_DEPTH = 2

// GraphQLType represents a simplified view of a GraphQL type.
type GraphQLType map[string]interface{}

// GeneratedFieldQuery represents a generated query template for a given field.
type GeneratedFieldQuery struct {
	Field   string `json:"field"`
	Query   string `json:"query"`
}

// LoadIntrospection reads the introspection JSON file and returns the __schema.
func LoadIntrospection(filePath string) (GraphQLType, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading introspection file: %w", err)
	}
	var introspection map[string]interface{}
	if err := json.Unmarshal(data, &introspection); err != nil {
		return nil, fmt.Errorf("error unmarshalling introspection JSON: %w", err)
	}
	schema, ok := introspection["data"].(map[string]interface{})["__schema"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to locate __schema in introspection data")
	}
	return schema, nil
}

// BuildTypesIndex creates a map of type name to its definition.
func BuildTypesIndex(schema GraphQLType) map[string]GraphQLType {
	typesIndex := make(map[string]GraphQLType)
	if types, ok := schema["types"].([]interface{}); ok {
		for _, t := range types {
			if typeMap, ok := t.(map[string]interface{}); ok {
				if name, ok := typeMap["name"].(string); ok && name != "" {
					typesIndex[name] = typeMap
				}
			}
		}
	}
	return typesIndex
}

// isScalar returns true if the type is a SCALAR or ENUM.
func isScalar(t GraphQLType) bool {
	kind, _ := t["kind"].(string)
	return kind == "SCALAR" || kind == "ENUM"
}

// unwrapType recursively unwraps a type (removes NON_NULL and LIST wrappers).
func unwrapType(t GraphQLType) GraphQLType {
	for {
		if ofType, ok := t["ofType"].(map[string]interface{}); ok {
			t = ofType
		} else {
			break
		}
	}
	return t
}

// generatePlaceholder returns a placeholder string for a given argument type.
func generatePlaceholder(argType GraphQLType) string {
	base := unwrapType(argType)
	if name, _ := base["name"].(string); name != "" {
		switch name {
		case "ID":
			return `"PLACEHOLDER_ID"`
		case "String", "DateTime":
			return `"PLACEHOLDER"`
		case "Int", "Float":
			return "0"
		case "Boolean":
			return "true"
		default:
			return `"PLACEHOLDER"`
		}
	}
	return `"PLACEHOLDER"`
}

// generateArgsString builds a string for field arguments using placeholders.
func generateArgsString(args []interface{}) string {
	var parts []string
	for _, a := range args {
		arg, ok := a.(map[string]interface{})
		if !ok {
			continue
		}
		argName := arg["name"].(string)
		placeholder := generatePlaceholder(arg["type"].(map[string]interface{}))
		parts = append(parts, fmt.Sprintf("%s: %s", argName, placeholder))
	}
	if len(parts) > 0 {
		return "(" + strings.Join(parts, ", ") + ")"
	}
	return ""
}

// generateSelectionSet recursively creates a selection set for an object type.
func generateSelectionSet(t GraphQLType, typesIndex map[string]GraphQLType, depth int) string {
	if depth >= MAX_DEPTH {
		return ""
	}
	base := unwrapType(t)
	if isScalar(base) {
		return ""
	}
	// Get the type definition from index.
	typeName, _ := base["name"].(string)
	typeDef, ok := typesIndex[typeName]
	if !ok {
		return ""
	}
	fields, ok := typeDef["fields"].([]interface{})
	if !ok {
		return ""
	}
	var selections []string
	for _, f := range fields {
		field, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		// Only include fields that are scalar or enum.
		fieldType := unwrapType(field["type"].(map[string]interface{}))
		if isScalar(fieldType) {
			selections = append(selections, field["name"].(string))
		}
	}
	if len(selections) > 0 {
		return "{ " + strings.Join(selections, " ") + " }"
	}
	return ""
}

// generateQueryForField creates a sample query for a given field.
func generateQueryForField(field map[string]interface{}, typesIndex map[string]GraphQLType, root string) string {
	args, _ := field["args"].([]interface{})
	argsStr := generateArgsString(args)
	selectionSet := ""
	if !isScalar(unwrapType(field["type"].(map[string]interface{}))) {
		selectionSet = generateSelectionSet(field["type"].(map[string]interface{}), typesIndex, 1)
	}
	queryBody := fmt.Sprintf("%s%s %s", field["name"].(string), argsStr, selectionSet)
	return fmt.Sprintf("%s { %s }", root, queryBody)
}

// GenerateQueries scans the Query and Mutation types and generates sample queries.
func GenerateQueries(schema GraphQLType) map[string][]GeneratedFieldQuery {
	typesIndex := BuildTypesIndex(schema)
	generated := map[string][]GeneratedFieldQuery{
		"query":    {},
		"mutation": {},
	}

	// Generate for Query type.
	if queryTypeName, ok := schema["queryType"].(map[string]interface{})["name"].(string); ok {
		if queryType, exists := typesIndex[queryTypeName]; exists {
			if fields, ok := queryType["fields"].([]interface{}); ok {
				for _, f := range fields {
					if field, ok := f.(map[string]interface{}); ok {
						q := generateQueryForField(field, typesIndex, "query")
						generated["query"] = append(generated["query"], GeneratedFieldQuery{
							Field: field["name"].(string),
							Query: q,
						})
					}
				}
			}
		}
	}

	// Generate for Mutation type if available.
	if mutationTypeName, ok := schema["mutationType"].(map[string]interface{})["name"].(string); ok {
		if mutationType, exists := typesIndex[mutationTypeName]; exists {
			if fields, ok := mutationType["fields"].([]interface{}); ok {
				for _, f := range fields {
					if field, ok := f.(map[string]interface{}); ok {
						q := generateQueryForField(field, typesIndex, "mutation")
						generated["mutation"] = append(generated["mutation"], GeneratedFieldQuery{
							Field: field["name"].(string),
							Query: q,
						})
					}
				}
			}
		}
	}

	return generated
}
