package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/vodafon/rawhttp"
)

var (
	flagProxy    = flag.String("x", "http://127.0.0.1:8080", "proxy")
	flagChangeCL = flag.Bool("cl", true, "recalculate content length")
	flagDebug    = flag.Bool("debug", false, "debug mode")
)

type Input struct {
	URL     string
	Request []byte
}

func main() {
	flag.Parse()

	// io.ReadAll reads until EOF (Ctrl+D or pipe close)
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read stdin: %v\n", err)
		os.Exit(1)
	}
	if *flagDebug {
		fmt.Printf("Data: %q\n", data)
	}

	input, err := ParseData(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid input: %v\n", err)
		os.Exit(1)
	}
	if *flagChangeCL {
		input.Request = replaceContentLength(input.Request)
	}

	client := rawhttp.NewDefaultClient()
	defer client.Close()

	if *flagProxy != "" {
		u, err := url.Parse(*flagProxy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to proxy URL: %v\n", err)
			os.Exit(1)
		}

		client.SetProxy(u)
	}

	req := &rawhttp.Request{}
	resp := &rawhttp.Response{}

	req.URL = input.URL
	req.Rawdata = input.Request

	if *flagDebug {
		fmt.Printf("%q\n", req.Rawdata)
	}

	err = client.Do(req, resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "do request error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", resp.Bytes())
}

func ParseData(data []byte) (*Input, error) {
	if bytes.HasPrefix(data, []byte("#")) {
		return ParseDataWithTarget(data)
	}

	host := getHostRegex(data)

	if host == "" {
		return nil, fmt.Errorf("target line and host header missed, cant understand the target")
	}

	input := &Input{}
	input.URL = "https://" + host
	input.Request = dataNormalization(data)

	return input, nil
}

func ParseDataWithTarget(data []byte) (*Input, error) {
	// 1. Split on the first newline to separate the URL line from the request
	// We use \n because even if the request uses \r\n, the first line usually
	// ends with a simple newline in these custom formats.
	parts := bytes.SplitN(data, []byte("\n"), 2)

	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format: could not find request body after URL line")
	}

	input := &Input{}

	// 2. Extract and clean the URL (u)
	// Remove '#' and any surrounding whitespace
	input.URL = string(bytes.TrimSpace(bytes.TrimPrefix(parts[0], []byte("#"))))
	req := parts[1]
	delimiter := []byte("\n\n")
	// 1. Check if it contains the sequence
	if !bytes.Contains(req, delimiter) {
		// 2. Trim leading/trailing whitespace
		req = bytes.TrimSpace(req)

		// 3. Add to end
		req = append(req, delimiter...)
	}
	input.Request = dataNormalization(req)

	return input, nil
}

func dataNormalization(req []byte) []byte {
	req = replaceLastCR(req)
	req = bytes.ReplaceAll(req, []byte("\r\n"), []byte("\n"))
	req = bytes.ReplaceAll(req, []byte("\n"), []byte("\r\n"))

	return req
}

func replaceLastCR(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] != '\r' {
		return data
	}
	result := make([]byte, len(data))
	copy(result, data)
	result[len(result)-1] = '\n'
	return result
}

func replaceContentLength(rawRequest []byte) []byte {
	re := regexp.MustCompile(`(?i)Content-Length:\s*\d+`)
	return re.ReplaceAll(rawRequest, []byte("Content-Length: ||CLEN||"))
}

func getHostRegex(rawReq []byte) string {
	// Convert bytes to string for regex processing
	reqStr := string(rawReq)

	// Regex Explanation:
	// (?m) - Multi-line mode: ^ and $ match start/end of line
	// (?i) - Case-insensitive: matches 'Host:', 'host:', 'HOST:', etc.
	// ^host: - Start of line followed by "host:"
	// \s* - Allow optional whitespace after colon
	// (.*) - Capture the rest of the line (the value)
	// $ - End of line
	re := regexp.MustCompile(`(?mi)^host:\s*(.*)$`)

	matches := re.FindStringSubmatch(reqStr)

	// matches[0] is the full match, matches[1] is the capture group
	if len(matches) > 1 {
		// TrimSpace removes trailing \r (CR) and potential spaces
		return strings.TrimSpace(matches[1])
	}

	return ""
}
