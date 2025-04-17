package types

import "time"

// CLI types
type CLIConfig struct {
	ConfigFile    string
	BaseURL       string
	Detect        bool
	OutputFile    string
	Timeout       time.Duration
	LogLevel      string
	LogFile       string
	NoColor       bool
	MaxDepth      int
	SchemaFile    string
	List          string
	Query         string
	Mutation      string
	AllQueries    bool
	AllMutations  bool
	Subscribe     bool
	SubQuery      string
	WSURL         string
	Execute       bool
	QueryString   string
	QueryFile     string
	Variables     string
	VariablesFile string
	Headers       map[string]string
}

type FileConfig struct {
	BaseURL    string            `yaml:"base" json:"base"`
	Detect     bool              `yaml:"detect" json:"detect"`
	TimeoutRaw string            `yaml:"timeout" json:"timeout"`
	Timeout    time.Duration     `yaml:"-" json:"-"`
	LogLevel   string            `yaml:"log-level" json:"log-level"`
	LogFile    string            `yaml:"log-file" json:"log-file"`
	Headers    map[string]string `yaml:"headers" json:"headers"`
	NoColor    bool              `yaml:"no-color" json:"no-color"`
	SchemaFile string            `yaml:"schema-file" json:"schema-file"`
	OutputFile string            `yaml:"output" json:"output"`
	MaxDepth   int               `yaml:"max-depth" json:"max-depth"`
}

// GraphQLRequest represents a GraphQL request structure.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// GraphQLError represents a single GraphQL error.
type GraphQLError struct {
	Message string `json:"message"`
}

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

// String returns a string representation of the TypeRef
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
