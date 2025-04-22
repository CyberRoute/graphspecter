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
├── config.yml
├── go.mod
├── go.sum
├── img
│   └── graphspecter.png
├── LICENSE
├── main.go
├── pkg
│   ├── cli
│   │   └── cli.go
│   ├── cmd
│   │   └── root.go
│   ├── config
│   │   ├── config.go
│   │   └── merge.go
│   ├── introspection
│   │   └── introspection.go
│   ├── logger
│   │   └── logger.go
│   ├── network
│   │   └── client.go
│   ├── schema
│   │   └── schema.go
│   ├── subscription
│   │   └── client.go
│   └── types
│       └── types.go
├── README.md

```

## Usage

```
# Run in detection mode
go run main.go --base http://192.168.1.1:5013 --detect

# Execute a single query or mutation
go run main.go \
  --execute \
  --base http://your.server/graphql \
  --query-string 'query { users { id name } }'

# Execute from files
go run main.go \
  --execute \
  --base http://your.server/graphql \
  --query-file getUser.graphql \
  --vars-file getUser.json

# Batch execution of all ops in 'ops' directory
# (expects pairs: *.graphql + optional *.json vars)
go run main.go \
  --batch-dir ./ops \
  --base http://your.server/graphql
```

### Options
```
  Usage of:

  -all-mutations                Print all mutations
  -all-queries                  Print all queries
  -base string                  Base URL of the target (e.g. http://192.168.1.1:5013)
  -batch-dir string             Directory of .graphql/.json pairs to execute in bulk (batch mode)
  -config string                Path to config file (.yaml or .json)
  -detect                       Enable detection mode to find a GraphQL endpoint
  -execute                      Execute a query or mutation
  -list string                  List queries, mutations or both (valid: 'queries', 'mutations', 'all')
  -log-file string              Log to file in addition to stdout
  -log-level string             Log level (debug, info, warn, error)
  -max-depth int                Maximum depth for selection sets (default 10)
  -mutation string              Print named mutations (comma-separated)
  -no-color                     Disable colored output
  -output string                Dump introspection schema (default "introspection_<endpoint>.json")
  -query string                 Print named queries (comma-separated)
  -query-file string            Path to file containing GraphQL query
  -query-string string          GraphQL query string to execute
  -schema-file string           File with the GraphQL schema (introspection JSON)
  -sub-query string             Subscription query to execute
  -subscribe                    Enable subscription mode
  -timeout duration             Timeout for operations (e.g., 30s, 1m) (default 1s)
  -vars string                  Query variables as JSON string
  -vars-file string             Path to JSON file with variables
  -ws-url string                WebSocket URL for subscriptions (default "ws://192.168.1.100:5013/subscriptions")
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
- The injections tests in `./ops` are run against https://github.com/dolevf/Damn-Vulnerable-GraphQL-Application
