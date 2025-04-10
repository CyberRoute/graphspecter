<p align="center">
   <img alt="GraphSpecter" src="img/graphspecter.png" width="200" height="200"/>
   <p align="center">
   </p>
 </p>

## Features

- Check if GraphQL introspection is enabled
- Export introspection data to JSON file
- Exports queries and mutations ready to test

## Project Structure

```
graphspecter
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── img
│   └── graphspecter.png
├── main.go
└── pkg
    ├── cli
    │   └── cli.go
    ├── introspection
    │   └── introspection.go
    ├── logger
    │   └── logger.go
    ├── network
    │   └── client.go
    ├── schema
    │   └── schema.go
    └── types
        └── types.go
```

## Usage

```
go run main.go -base http://192.168.1.1:5013 -detect
```

### Options
```
  -all-mutations
    	Only print mutations (by default both queries and mutations are printed)
  -all-queries
    	Only print queries (by default both queries and mutations are printed)
  -base string
    	Base URL of the target (e.g. http://192.168.1.1:5013)
  -detect
    	Enable detection mode to find a GraphQL endpoint
  -list string
    	Parse GraphQL schema and list queries, mutations or both (valid values: 'queries', 'mutations' or 'all')
  -log-file string
    	Log to file in addition to stdout
  -log-level string
    	Log level (debug, info, warn, error) (default "info")
  -max-depth int
    	Set the maximum level of nested field traversal when generating GraphQL selection sets from the introspection schema (default 10)
  -mutation string
    	Only print named mutations (comma-separated list of mutation names)
  -no-color
    	Disable colored output
  -output string
    	Output file for introspection results (default "introspection.json")
  -query string
    	Only print named queries (comma-separated list of query names)
  -schema-file string
    	File with the GraphQL schema (introspection JSON)
  -timeout duration
    	Timeout for operations (e.g., 30s, 1m) (default 1s)
```
## Building

```
go build -o graphspecter
```

## Example

```
# Check if introspection is enabled
./graphspecter -base http://192.168.1.1:5013 -detect -output results.json
```

## Authentication

You can authenticate requests by setting the `AUTH_TOKEN` environment variable. When set, all requests will include an `Authorization: Bearer <token>` header.

Example:
```
# Set the authentication token
export AUTH_TOKEN="your-token-here"
```

## Security Notes

- GraphQL introspection is a feature that allows clients to query a GraphQL server for information about its schema.
- While useful for development, introspection should typically be disabled in production environments as it may expose sensitive information about your API structure.
