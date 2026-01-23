package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/vodafon/rawhttp"
)

var flagProxy = flag.String("x", "http://127.0.0.1:8080", "proxy")

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

	input, err := ParseData(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid input: %v\n", err)
		os.Exit(1)
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

	//	fmt.Printf("%q\n", req.Rawdata)

	err = client.Do(req, resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "do request error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", resp.Bytes())
}

func ParseData(data []byte) (*Input, error) {
	if !bytes.HasPrefix(data, []byte("#")) {
		return nil, fmt.Errorf("invalid input format. target line not found")
	}

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
	delimiter := []byte("\r\n\r\n")
	// 1. Check if it contains the sequence
	if !bytes.Contains(req, delimiter) {
		// 2. Trim leading/trailing whitespace
		req = bytes.TrimSpace(req)

		// 3. Add to end
		req = append(req, delimiter...)
	}

	// 3. Extract the raw request (request)
	input.Request = req

	return input, nil
}
