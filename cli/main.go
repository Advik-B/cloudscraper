package main

import (
	"io"
	"log"
	"os"

	"github.com/Advik-B/cloudscraper/lib"
	"github.com/Advik-B/cloudscraper/lib/js"
)

func main() {
	// Enable debug logging to see the library's operations.
	logger := log.New(os.Stdout, "cloudscraper: ", log.LstdFlags)

	const jsRuntime = js.Node
	logger.Println("Attempting to create a scraper that uses", jsRuntime)

	// Create a new scraper instance, specifically configuring it to use "node".
	// The library will automatically find the 'node' executable in your system's PATH.
	sc, err := cloudscraper.New(
		cloudscraper.WithJSRuntime(jsRuntime),
		cloudscraper.WithLogger(logger),
	)
	if err != nil {
		// This error will trigger if 'node' is not found in the PATH.
		logger.Fatalf("Failed to create scraper: %v. Is Node.js installed and in your PATH?", err)
	}

	logger.Println("Scraper created successfully. Making request...")

	targetURL := "https://www.crunchbase.com/organization/openai"

	// Make a GET request
	resp, err := sc.Get(targetURL)
	if err != nil {
		log.Fatalf("Request to %s failed: %v", targetURL, err)
	}
	defer resp.Body.Close()

	logger.Printf("\n--- Response ---\n")
	logger.Printf("Status: %s\n", resp.Status)

	// Read and print the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Fatalf("Failed to read response body: %v", err)
	}

	// Print a preview of the HTML to confirm success
	preview := string(body)
	if len(preview) > 500 {
		preview = preview[:500]
	}
	logger.Printf("Body Preview:\n%s...\n", preview)
	logger.Println("----------------")

	if resp.StatusCode == 200 {
		logger.Println("\nSuccess! Cloudflare challenge was bypassed using ", jsRuntime)
	} else {
		logger.Printf("\nFailed to bypass challenge. Received status code: %d\n", resp.StatusCode)
	}
}
