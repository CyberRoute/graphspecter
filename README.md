<table width="100%" align="center">
  <tr>
    <td align="center" bgcolor="gray">
      <img src="img/graphspecter.png" width="200" style="border-radius: 50%;">
      <h1 style="color: white;">A GraphQL security auditing tool</h1>
    </td>
  </tr>
</table>

## Features

- Check if GraphQL introspection is enabled
- Export introspection data to JSON file
- Fingerprint GraphQL engine (Apollo, and more to come)
- Run customized queries against endpoints

## Project Structure

```
graphspecter/
├── go.mod
├── introspection.json
├── main.go
├── pkg
│   ├── fingerprint
│   │   └── fingerprint.go
│   ├── generator
│   │   └── query_generator.go
│   ├── introspection
│   │   └── introspection.go
│   ├── network
│   │   └── client.go
│   └── types
│       └── types.go
└── README.md
```

## Usage

```
go run main.go -base http://192.168.1.1:5013 -detect
```

### Options

- `-base`: Base URL of the target (required)
- `-endpoint`: Specific GraphQL endpoint (if not provided, detection will be attempted)
- `-detect`: Enable detection mode to scan common endpoints
- `-fingerprint`: Enable GraphQL engine fingerprinting
- `-output`: Output file for introspection results (default: "introspection.json")

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

# Run the tool with authentication
./graphspecter -base http://192.168.1.1:5013 -endpoint /graphql
```

## Security Notes

- GraphQL introspection is a feature that allows clients to query a GraphQL server for information about its schema.
- While useful for development, introspection should typically be disabled in production environments as it may expose sensitive information about your API structure.
