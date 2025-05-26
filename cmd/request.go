package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/spf13/cobra"
)

func NewRequestCmd(opts *CommandOptions) *cobra.Command {
	requestCmd := &cobra.Command{
		Use:   "request",
		Short: "Generate mock data based from an object string or from template files",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			method, _ := cmd.Flags().GetString("method")
			forceHttps, _ := cmd.Flags().GetBool("https")
			urlStr, _ := cmd.Flags().GetString("url")
			headers, _ := cmd.Flags().GetStringSlice("header")
			data, _ := cmd.Flags().GetString("data")
			queryParams, _ := cmd.Flags().GetStringSlice("qs")
			//responseAccessor := cmd.Flags().GetString("response-accessor")
			withMetrics, _ := cmd.Flags().GetBool("with-metrics")
			onlyResponseBody, _ := cmd.Flags().GetBool("only-response-body")

			mocker := mocker.New()
			method = strings.ToUpper(method)

			// Validate flags
			if urlStr == "" {
				return fmt.Errorf("empty url, ask for help -h or --help")
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
				return fmt.Errorf("error parsing URL '%w'", err)
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
				var parseMap map[string]any
				if err := json.Unmarshal([]byte(data), &parseMap); err != nil {
					return fmt.Errorf("failed to parse JSON from the provided --data '%w'", err)
				}

				// Process the parsed map
				if err := processJsonMap(parseMap, mocker); err != nil {
					return fmt.Errorf("%w", err)
				}

				// Sanitize the parsed map
				sanitizeJsonMap(parseMap)

				// Convert back to JSON string
				jsonBytes, err := json.Marshal(parseMap)
				if err != nil {
					return fmt.Errorf("failed to convert JSON map to string '%w'", err)
				}

				body = bytes.NewBuffer(jsonBytes)
			}

			// Create request
			req, err := http.NewRequest(method, parsedUrl.String(), body)
			if err != nil {
				return fmt.Errorf("error creating request '%w'", err)
			}

			// Add request headers
			for _, header := range headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header format '%s'", header)
				}
				req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}

			// Execute request
			client := &http.Client{}
			startTime := time.Now()
			resp, err := client.Do(req)
			duration := time.Since(startTime)
			if err != nil {
				return fmt.Errorf("error executing request '%w'", err)
			}
			defer resp.Body.Close()

			// Read response body
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("error reading response body '%w'", err)
			}

			if !onlyResponseBody {
				fmt.Fprintf(opts.Out, "Status: %s\n", resp.Status)
				if withMetrics {
					fmt.Fprintf(opts.Out, "Metrics:\n")
					fmt.Fprintf(opts.Out, "  Duration:%s\n", formatDurationMetrics(duration))
					fmt.Fprintf(opts.Out, "  Size:%s\n", formatSizeMetrics(int64(len(respBody))))
				}
				fmt.Fprintf(opts.Out, "URL: %s\n", req.URL.String())
				fmt.Fprintf(opts.Out, "Headers:\n")
				for k, v := range resp.Header {
					fmt.Fprintf(opts.Out, "  %s: %s\n", k, strings.Join(v, ", "))
				}
				fmt.Fprintf(opts.Out, "Body:\n")
			}
			fmt.Fprintf(opts.Out, "%s", string(respBody))

			return nil
		},
	}

	requestCmd.Flags().String("method", "GET", "the method to be used in the request (e.g. GET, POST, PUT, DELETE)")
	requestCmd.Flags().Bool("https", false, "if set, use https instead of http")
	requestCmd.Flags().String("url", "", "the url of the request, with added Url params (e.g. localhost:8080, localhost:8000/api/v1/users, api.com/user/{UUID.uuidv4})")
	requestCmd.Flags().StringArray("header", []string{}, "pass a string 'header', in key:value format, to be used as the request header, (e.g. 'Authorization: Bearer {token}')")
	requestCmd.Flags().String("data", "", "pass a JSON object as a string to be used as the request body")
	requestCmd.Flags().StringArray("qs", []string{}, "pass a string 'query string' to be used as the request query string")
	requestCmd.Flags().String("response-accessor", "", "pass a string 'response accessor', (e.g. ['token']), to be used to access the response data, if unable to access the data, the whole response will be returned")
	requestCmd.Flags().Bool("with-metrics", false, "if set, show metrics of the request")
	requestCmd.Flags().Bool("only-response-body", false, "if set, the command output will be only the response's body, nothing more")

	// Configure cobra ouput streams to use the custom 'Out'
	requestCmd.SetOut(opts.Out)

	return requestCmd
}
