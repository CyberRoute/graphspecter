package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/yourusername/black_hat_gql/pkg/fingerprint"
	"github.com/yourusername/black_hat_gql/pkg/generator"
	"github.com/yourusername/black_hat_gql/pkg/introspection"
	"github.com/yourusername/black_hat_gql/pkg/network"
)

func main() {
	// Flags:
	// -base: Base URL (e.g. "http://192.168.1.1:5013")
	// -endpoint: Specific GraphQL endpoint (e.g. "/graphql")
	// -detect: Enable detection mode (scan common endpoints)
	// -fingerprint: Enable GraphQL engine fingerprinting
	// -output: File to save introspection results
	baseURL := flag.String("base", "", "Base URL of the target (e.g. http://192.168.1.1:5013)")
	endpoint := flag.String("endpoint", "", "Specific GraphQL endpoint (if not provided, detection will be attempted)")
	detect := flag.Bool("detect", false, "Enable detection mode to find a GraphQL endpoint")
	fingerprintFlag := flag.Bool("fingerprint", false, "Enable GraphQL engine fingerprinting")
	outputFile := flag.String("output", "introspection.json", "Output file for introspection results")
	generateQueriesFlag := flag.Bool("generateQueries", false, "Generate query templates for all enum-based arguments using introspection data")
	introspectionPath := flag.String("introspection", "introspection.json", "Path to the introspection JSON file")
	queryOutputPath := flag.String("queryOutput", "generated_queries.json", "Output file for generated queries")

	flag.Parse()

	if *generateQueriesFlag {
		schema, err := generator.LoadIntrospection(*introspectionPath)
		if err != nil {
			fmt.Printf("[-] Error loading introspection: %v\n", err)
			os.Exit(1)
		}
		queries := generator.GenerateQueries(schema)
		outputJSON, err := json.MarshalIndent(queries, "", "  ")
		if err != nil {
			fmt.Printf("[-] Error marshalling generated queries: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*queryOutputPath, outputJSON, 0644); err != nil {
			fmt.Printf("[-] Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[+] Generated queries have been written to %s\n", *queryOutputPath)
		os.Exit(0)
	}
	

	if *baseURL == "" {
		fmt.Println("Error: Base URL is required.")
		flag.Usage()
		os.Exit(1)
	}

	var targetURL string
	if *endpoint != "" {
		targetURL = *baseURL + *endpoint
		fmt.Printf("[*] Using provided endpoint: %s\n", targetURL)
	} else if *detect {
		fmt.Println("[*] Detection mode enabled. Scanning for GraphQL endpoint...")
		detectedEndpoint, err := network.DetectGraphQLEndpoint(*baseURL)
		if err != nil {
			fmt.Printf("[-] Detection failed: %v\n", err)
			os.Exit(1)
		}
		targetURL = detectedEndpoint
	} else {
		// Use the base URL directly if no endpoint/detection is provided.
		targetURL = *baseURL
		fmt.Printf("[*] Using base URL as target: %s\n", targetURL)
	}

	fmt.Println("[+] Starting GraphQL security audit...")
	fmt.Printf("[+] Target: %s\n", targetURL)

	// Common headers for all requests
	headers := map[string]string{"Content-Type": "application/json"}
	
	// Add Authorization header if AUTH_TOKEN environment variable is set
	if authToken := os.Getenv("AUTH_TOKEN"); authToken != "" {
		headers["Authorization"] = "Bearer " + authToken
	}

	// If fingerprint flag is enabled, attempt to detect the GraphQL engine.
	if *fingerprintFlag {
		engine, err := fingerprint.DetectEngine(targetURL, headers)
		if err != nil {
			fmt.Println("[*] Could not determine GraphQL engine.")
		} else {
			fmt.Printf("[*] Discovered GraphQL Engine: %s\n", engine)
		}
	}
	// Check introspection.
	fmt.Println("[+] Checking if introspection is enabled...")
	introspectionResult, err := introspection.CheckIntrospection(targetURL, headers)
	if err != nil {
		fmt.Printf("[-] Error checking introspection: %v\n", err)
		os.Exit(1)
	}

	// Write introspection result to file.
	err = introspection.WriteIntrospectionToFile(introspectionResult, *outputFile)
	if err != nil {
		fmt.Printf("[-] Error writing introspection result to file: %v\n", err)
		os.Exit(1)
	}

	if introspection.IsIntrospectionEnabled(introspectionResult) {
		fmt.Println("[!] WARNING: Introspection is ENABLED!")
		fmt.Printf("[+] Introspection data saved to %s\n", *outputFile)
	} else {
		fmt.Println("[+] Introspection appears to be disabled.")
	}

	fmt.Println("[+] Audit completed.")
}