package schema

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CyberRoute/graphspecter/pkg/logger"
)

// TypeKind represents the different kinds of GraphQL types
type TypeKind string

const (
	// Type kind constants
	SCALAR       TypeKind = "SCALAR"
	OBJECT       TypeKind = "OBJECT"
	INTERFACE    TypeKind = "INTERFACE"
	UNION        TypeKind = "UNION"
	ENUM         TypeKind = "ENUM"
	INPUT_OBJECT TypeKind = "INPUT_OBJECT"
	LIST         TypeKind = "LIST"
	NON_NULL     TypeKind = "NON_NULL"
)

// Field represents a GraphQL field with its arguments and type information
type Field struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Args              []InputValue `json:"args"`
	Type              TypeRef      `json:"type"`
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason string       `json:"deprecationReason"`
}

// InputValue represents an input argument or field
type InputValue struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Type         TypeRef `json:"type"`
	DefaultValue string  `json:"defaultValue"`
}

// TypeRef represents a type reference, which can be nested for things like [String!]!
type TypeRef struct {
	Kind   TypeKind `json:"kind"`
	Name   string   `json:"name"`
	OfType *TypeRef `json:"ofType"`
}

// EnumValue represents a value in an enum type
type EnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}

// Type represents a GraphQL type in the schema
type Type struct {
	Kind          TypeKind     `json:"kind"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Fields        []Field      `json:"fields"`
	InputFields   []InputValue `json:"inputFields"`
	Interfaces    []TypeRef    `json:"interfaces"`
	EnumValues    []EnumValue  `json:"enumValues"`
	PossibleTypes []TypeRef    `json:"possibleTypes"`
}

// SchemaType represents a top-level schema type (query, mutation, subscription)
type SchemaType struct {
	Name string `json:"name"`
}

// Schema represents the top-level GraphQL schema
type Schema struct {
	QueryType        SchemaType  `json:"queryType"`
	MutationType     SchemaType  `json:"mutationType"`
	SubscriptionType SchemaType  `json:"subscriptionType"`
	Types            []Type      `json:"types"`
	Directives       []Directive `json:"directives"`
}

// Directive represents a GraphQL directive
type Directive struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Locations   []string     `json:"locations"`
	Args        []InputValue `json:"args"`
}

// IntrospectionResponse represents the full response from an introspection query
type IntrospectionResponse struct {
	Data struct {
		Schema Schema `json:"__schema"`
	} `json:"data"`
}

// GQLSchema is the main struct that holds the parsed schema information
type GQLSchema struct {
	Types    map[string]Type
	Query    *Type
	Mutation *Type
}

// LoadFromFile loads a GraphQL schema from an introspection result JSON file
func LoadFromFile(filePath string) (*GQLSchema, error) {
	logger.Info("Loading schema from file: %s", filePath)

	// Read file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var response IntrospectionResponse
	if err := json.Unmarshal(fileContent, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create and initialize schema
	schema := &GQLSchema{
		Types: make(map[string]Type),
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
func unwrapType(tr *TypeRef) *TypeRef {
	for tr.Kind == NON_NULL || tr.Kind == LIST {
		tr = tr.OfType
	}
	return tr
}

func (tr *TypeRef) String() string {
	if tr == nil {
		return ""
	}
	switch tr.Kind {
	case NON_NULL:
		return tr.OfType.String() + "!"
	case LIST:
		return "[" + tr.OfType.String() + "]"
	default:
		return tr.Name
	}
}

// generateSelectionSetWithCount recursively generates a selection set using a count-based cycle detection.
func generateSelectionSetWithCount(s *GQLSchema, typeName string, maxDepth int, indent string, visited map[string]int) string {
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
		if underlying.Kind == OBJECT {
			nested := generateSelectionSetWithCount(s, underlying.Name, maxDepth-1, newIndent, visited)
			if nested != "" {
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

// And modify GenerateQuery to use this new helper:
func (s *GQLSchema) GenerateQuery(fieldName string) (string, error) {
	if s.Query == nil {
		return "", fmt.Errorf("schema has no query type")
	}

	var queryField *Field
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
		query += " {\n    # Selection set would go here\n  }\n}"
	}
	return query, nil
}

// Modified GenerateMutation function to include a nested selection set using recursive logic.
func (s *GQLSchema) GenerateMutation(fieldName string) (string, error) {
	if s.Mutation == nil {
		return "", fmt.Errorf("schema has no mutation type")
	}

	// Find the specified mutation field.
	var mutationField *Field
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
func (s *GQLSchema) ListQueries() []string {
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
func (s *GQLSchema) ListMutations() []string {
	var mutations []string

	if s.Mutation == nil {
		return mutations
	}

	for _, field := range s.Mutation.Fields {
		mutations = append(mutations, field.Name)
	}

	return mutations
}
