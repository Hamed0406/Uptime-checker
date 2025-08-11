package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	api := os.Getenv("API_BASE")
	if api == "" {
		api = "http://localhost:8080"
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter a site URL to monitor (e.g., https://example.com): ")
	raw, _ := reader.ReadString('\n')
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	if _, err := url.ParseRequestURI(raw); err != nil {
		fmt.Println("Invalid URL.")
		return
	}

	body, _ := json.Marshal(map[string]string{"url": raw})
	resp, err := http.Post(api+"/api/targets", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error contacting API:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Added! Check the API logs and GET /api/targets.")
	} else {
		fmt.Println("API returned status:", resp.Status)
	}
}
