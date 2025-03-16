package types

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