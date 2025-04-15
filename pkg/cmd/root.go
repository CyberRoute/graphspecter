package cmd

import (
	"flag"
	"github.com/CyberRoute/graphspecter/pkg/types"
	"time"
)

func ParseFlags() *types.CLIConfig {
	cfg := &types.CLIConfig{}

	flag.StringVar(&cfg.BaseURL, "base", "", "Base URL of the target (e.g. http://192.168.1.1:5013)")
	flag.BoolVar(&cfg.Detect, "detect", false, "Enable detection mode to find a GraphQL endpoint")
	flag.StringVar(&cfg.OutputFile, "output", "introspection.json", "Output file for introspection results")
	flag.DurationVar(&cfg.Timeout, "timeout", 1*time.Second, "Timeout for operations (e.g., 30s, 1m)")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.LogFile, "log-file", "", "Log to file in addition to stdout")
	flag.BoolVar(&cfg.NoColor, "no-color", false, "Disable colored output")
	flag.IntVar(&cfg.MaxDepth, "max-depth", 10, "Maximum depth for selection sets")
	flag.StringVar(&cfg.SchemaFile, "schema-file", "", "File with the GraphQL schema (introspection JSON)")
	flag.StringVar(&cfg.List, "list", "", "List queries, mutations or both (valid: 'queries', 'mutations', 'all')")
	flag.StringVar(&cfg.Query, "query", "", "Print named queries (comma-separated)")
	flag.StringVar(&cfg.Mutation, "mutation", "", "Print named mutations (comma-separated)")
	flag.BoolVar(&cfg.AllQueries, "all-queries", false, "Print all queries")
	flag.BoolVar(&cfg.AllMutations, "all-mutations", false, "Print all mutations")
	flag.BoolVar(&cfg.Subscribe, "subscribe", false, "Enable subscription mode")
	flag.StringVar(&cfg.SubQuery, "sub-query", "", "Subscription query to execute")
	flag.StringVar(&cfg.WSURL, "ws-url", "ws://192.168.1.100:5013/subscriptions", "WebSocket URL for subscriptions")

	// Placeholder for future use
	flag.BoolVar(&cfg.Execute, "execute", false, "Execute a query or mutation (future feature)")
	flag.StringVar(&cfg.QueryString, "query-string", "", "GraphQL query string to execute")
	flag.StringVar(&cfg.QueryFile, "query-file", "", "Path to file containing GraphQL query")
	flag.StringVar(&cfg.Variables, "vars", "", "Query variables as JSON string")
	flag.StringVar(&cfg.VariablesFile, "vars-file", "", "Path to JSON file with variables")

	flag.Parse()
	return cfg
}

