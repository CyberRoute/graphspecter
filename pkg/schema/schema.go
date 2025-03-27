package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/CyberRoute/graphspecter/pkg/logger"
	"github.com/CyberRoute/graphspecter/pkg/types"
)

// LoadFromFile loads a GraphQL schema from an introspection result JSON file
func LoadFromFile(filePath string) (*types.GQLSchema, error) {
	logger.Info("Loading schema from file: %s", filePath)

	// Read file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var response types.IntrospectionResponse
	if err := json.Unmarshal(fileContent, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create and initialize schema
	schema := &types.GQLSchema{
		Types: make(map[string]types.Type),
	}

	// Add all types to the map for easy lookup
	for _, t := range response.Data.Schema.Types {
		schema.Types[t.Name] = t
	}

	// Set query type
	queryTypeName := response.Data.Schema.QueryType.Name
	if queryType, ok := schema.Types[queryTypeName]; ok {
		schema.Query = &queryType
	} else {
		logger.Warn("Query type '%s' not found in schema", queryTypeName)
	}

	// Set mutation type if it exists
	mutationTypeName := response.Data.Schema.MutationType.Name
	if mutationTypeName != "" {
		if mutationType, ok := schema.Types[mutationTypeName]; ok {
			schema.Mutation = &mutationType
		}
	}

	logger.Info("Schema loaded successfully")
	return schema, nil
}

// Helper function to recursively unwrap NON_NULL and LIST wrappers
func unwrapType(tr *types.TypeRef) *types.TypeRef {
	for tr.Kind == types.NON_NULL || tr.Kind == types.LIST {
		tr = tr.OfType
	}
	return tr
}

// generateSelectionSetWithCount recursively generates a selection set using a count-based cycle detection.
func generateSelectionSetWithCount(s *types.GQLSchema, typeName string, maxDepth int, indent string, visited map[string]int) string {
	if maxDepth <= 0 {
		return fmt.Sprintf("\n%s!!! MAX RECURSION DEPTH REACHED !!!", indent)
	}

	// If this type appears too many times, stop recursing.
	if count, ok := visited[typeName]; ok && count >= 2 { // allow a type to appear up to 2 times
		return fmt.Sprintf("\n%s!!! MAX RECURSION LIMIT for %s reached !!!", indent, typeName)
	}

	// Increment the count for this type.
	visited[typeName]++
	defer func() {
		visited[typeName]--
	}()

	typeDef, ok := s.Types[typeName]
	if !ok || len(typeDef.Fields) == 0 {
		return ""
	}

	selectionSet := ""
	newIndent := indent + "    "
	for _, f := range typeDef.Fields {
		underlying := unwrapType(&f.Type)
		if underlying.Kind == types.OBJECT {
			nested := generateSelectionSetWithCount(s, underlying.Name, maxDepth-1, newIndent, visited)
			if nested != "" && !strings.Contains(nested, "MAX RECURSION") {
				selectionSet += fmt.Sprintf("\n%s%s { %s\n%s}", newIndent, f.Name, nested, newIndent)
			} else {
				selectionSet += fmt.Sprintf("\n%s%s", newIndent, f.Name)
			}
		} else {
			selectionSet += fmt.Sprintf("\n%s%s", newIndent, f.Name)
		}
	}
	return selectionSet
}

// GenerateQuery generates a GraphQL query for the specified field
func GenerateQuery(s *types.GQLSchema, fieldName string) (string, error) {
	if s.Query == nil {
		return "", fmt.Errorf("schema has no query type")
	}

	var queryField *types.Field
	for _, field := range s.Query.Fields {
		if field.Name == fieldName {
			queryField = &field
			break
		}
	}
	if queryField == nil {
		return "", fmt.Errorf("field '%s' not found in query type", fieldName)
	}

	query := fmt.Sprintf("query %s {\n  %s", fieldName, fieldName)
	if len(queryField.Args) > 0 {
		query += "("
		for i, arg := range queryField.Args {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("%s: %s", arg.Name, arg.Type.String())
		}
		query += ")"
	}

	underlying := unwrapType(&queryField.Type)
	visited := make(map[string]int)
	selectionSet := generateSelectionSetWithCount(s, underlying.Name, 10, "  ", visited)
	if selectionSet != "" {
		query += " {" + selectionSet + "\n  }\n}"
	} else {
		// No selection set is needed, simply close the query.
		query += "\n}"
	}
	return query, nil
}

// GenerateMutation generates a GraphQL mutation for the specified field
func GenerateMutation(s *types.GQLSchema, fieldName string) (string, error) {
	if s.Mutation == nil {
		return "", fmt.Errorf("schema has no mutation type")
	}

	// Find the specified mutation field.
	var mutationField *types.Field
	for _, field := range s.Mutation.Fields {
		if field.Name == fieldName {
			mutationField = &field
			break
		}
	}
	if mutationField == nil {
		return "", fmt.Errorf("field '%s' not found in mutation type", fieldName)
	}

	// Begin building the mutation.
	mutation := fmt.Sprintf("mutation %s {\n  %s", fieldName, fieldName)

	// Add arguments if any.
	if len(mutationField.Args) > 0 {
		mutation += "("
		for i, arg := range mutationField.Args {
			if i > 0 {
				mutation += ", "
			}
			mutation += fmt.Sprintf("%s: %s", arg.Name, arg.Type.String())
		}
		mutation += ")"
	}

	// Get the underlying type of the mutation field.
	underlying := unwrapType(&mutationField.Type)
	// Use the same recursive helper with cycle detection.
	visited := make(map[string]int)
	selectionSet := generateSelectionSetWithCount(s, underlying.Name, 10, "  ", visited)
	if selectionSet != "" {
		mutation += " {" + selectionSet + "\n  }\n}"
	} else {
		mutation += " {\n    # Selection set would go here\n  }\n}"
	}

	return mutation, nil
}

// ListQueries returns all query names in the schema
func ListQueries(s *types.GQLSchema) []string {
	var queries []string

	if s.Query == nil {
		return queries
	}

	for _, field := range s.Query.Fields {
		queries = append(queries, field.Name)
	}

	return queries
}

// ListMutations returns all mutation names in the schema
func ListMutations(s *types.GQLSchema) []string {
	var mutations []string

	if s.Mutation == nil {
		return mutations
	}

	for _, field := range s.Mutation.Fields {
		mutations = append(mutations, field.Name)
	}

	return mutations
}
