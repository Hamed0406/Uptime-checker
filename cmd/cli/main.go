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
		api = "http://127.0.0.1:8080"
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a site URL to monitor (or type 'exit' to quit): ")
		raw, _ := reader.ReadString('\n')
		raw = strings.TrimSpace(raw)

		if strings.EqualFold(raw, "exit") {
			fmt.Println("Exiting CLI...")
			break
		}

		if !strings.Contains(raw, "://") {
			raw = "https://" + raw
		}
		if _, err := url.ParseRequestURI(raw); err != nil {
			fmt.Println("Invalid URL.")
			continue
		}

		body, _ := json.Marshal(map[string]string{"url": raw})
		resp, err := http.Post(api+"/api/targets", "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Println("Error contacting API:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			fmt.Println("Added! Check the API logs and GET /api/targets.")
		} else {
			fmt.Println("API returned status:", resp.Status)
		}
	}
}
