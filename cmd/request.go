package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lfsc09/k-test-n-stress/mock"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Generate mock data based from an object string or from template files",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		method := strings.ToUpper(viper.GetString("method"))
		forceHttps := viper.GetBool("https")
		urlStr := viper.GetString("url")
		headers := viper.GetStringSlice("header")
		data := viper.GetString("data")
		queryParams := viper.GetStringSlice("qs")
		// responseAccessor := viper.GetString("responseAccessor")
		withMetrics := viper.GetBool("withMetrics")
		onlyResponseBody := viper.GetBool("onlyResponseBody")

		mocker := mock.New()

		// Validate flags
		if urlStr == "" {
			log.Fatalln("Empty url. Ask for help -h or --help")
		}

		// Decide the URL prefix
		var urlPrefix string
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			if forceHttps {
				urlPrefix = "https://"
			} else {
				urlPrefix = "http://"
			}
		}
		urlStr = urlPrefix + urlStr

		// Mock Url params if present
		urlStr = processStr(urlStr, mocker)

		// Parse the URL
		parsedUrl, err := url.Parse(urlStr)
		if err != nil {
			log.Fatalf("Error parsing URL: %v\n", err)
		}

		// Add query string if present
		query := parsedUrl.Query()
		for _, queryParam := range queryParams {
			parts := strings.SplitN(queryParam, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := processStr(strings.TrimSpace(parts[1]), mocker)
				query.Add(key, value)
			}
		}
		parsedUrl.RawQuery = query.Encode()

		// Create and mock request body if present
		var body io.Reader
		if data != "" {
			// Parse the string object content
			var parseMap map[string]interface{}
			if err := json.Unmarshal([]byte(data), &parseMap); err != nil {
				log.Fatalf("Failed to parse JSON from the provided --data <string>: %v\n", err)
			}

			// Process the parsed map
			if err := processJsonMap(parseMap, mocker); err != nil {
				log.Fatalln(err)
			}

			// Sanitize the parsed map
			sanitizeJsonMap(parseMap)

			// Convert back to JSON string
			jsonBytes, err := json.Marshal(parseMap)
			if err != nil {
				log.Fatalf("Failed to convert JSON map to string: %v\n", err)
			}

			body = bytes.NewBuffer(jsonBytes)
		}

		// Create request
		req, err := http.NewRequest(method, parsedUrl.String(), body)
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

		// Add request headers
		for _, header := range headers {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) != 2 {
				log.Fatalf("Invalid header format: %v\n", header)
			}
			req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}

		// Execute request
		client := &http.Client{}
		startTime := time.Now()
		resp, err := client.Do(req)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Error executing request: %v\n", err)
		}
		defer resp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error reading response body: %v\n", err)
		}

		if !onlyResponseBody {
			fmt.Printf("Status: %s\n", resp.Status)
			if withMetrics {
				fmt.Printf("Metrics:\n")
				fmt.Printf("  Duration: %s\n", formatDurationMetrics(duration))
				fmt.Printf("  Size: %s\n", formatSizeMetrics(int64(len(respBody))))
			}
			fmt.Printf("URL: %s\n", req.URL.String())
			fmt.Println("Headers:")
			for k, v := range resp.Header {
				fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
			}
			fmt.Println("\nBody:")
		}
		fmt.Println(string(respBody))
	},
}

func init() {
	requestCmd.Flags().String("method", "GET", "the method to be used in the request (e.g. GET, POST, PUT, DELETE)")
	requestCmd.Flags().Bool("https", false, "if set, use https instead of http")
	requestCmd.Flags().String("url", "", "the url of the request, with added Url params (e.g. localhost:8080, localhost:8000/api/v1/users, api.com/user/{UUID.uuidv4})")
	requestCmd.Flags().StringArray("header", []string{}, "pass a string 'header', in key:value format, to be used as the request header, (e.g. 'Authorization: Bearer {token}')")
	requestCmd.Flags().String("data", "", "pass a JSON object as a string to be used as the request body")
	requestCmd.Flags().StringArray("qs", []string{}, "pass a string 'query string' to be used as the request query string")
	requestCmd.Flags().String("responseAccessor", "", "pass a string 'response accessor', (e.g. ['token']), to be used to access the response data, if unable to access the data, the whole response will be returned")
	requestCmd.Flags().Bool("withMetrics", true, "if set, show metrics of the request")
	requestCmd.Flags().Bool("onlyResponseBody", false, "if set, the command output will be only the response's body, nothing more")

	viper.BindPFlag("method", requestCmd.Flags().Lookup("method"))
	viper.BindPFlag("https", requestCmd.Flags().Lookup("https"))
	viper.BindPFlag("url", requestCmd.Flags().Lookup("url"))
	viper.BindPFlag("header", requestCmd.Flags().Lookup("header"))
	viper.BindPFlag("data", requestCmd.Flags().Lookup("data"))
	viper.BindPFlag("qs", requestCmd.Flags().Lookup("qs"))
	viper.BindPFlag("responseAccessor", requestCmd.Flags().Lookup("responseAccessor"))
	viper.BindPFlag("withMetrics", requestCmd.Flags().Lookup("withMetrics"))
	viper.BindPFlag("onlyResponseBody", requestCmd.Flags().Lookup("onlyResponseBody"))

	rootCmd.AddCommand(requestCmd)
}

func formatDurationMetrics(duration time.Duration) string {
	switch {
	case duration < time.Millisecond:
		return fmt.Sprintf("[%.2fµs]", float64(duration.Microseconds()))
	case duration < time.Second:
		return fmt.Sprintf("[%.2fms]", float64(duration.Milliseconds()))
	default:
		return fmt.Sprintf("[%.2fs]", duration.Seconds())
	}
}

func formatSizeMetrics(size int64) string {
	switch {
	case size >= GB:
		return fmt.Sprintf("[%.2f GB]", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("[%.2f MB]", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("[%.2f KB]", float64(size)/float64(MB))
	default:
		return fmt.Sprintf("[%d Bytes]", size)
	}
}
