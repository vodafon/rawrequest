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
	"time"

	"github.com/vodafon/rawhttp"
	"github.com/vodafon/rawhttp2"
)

var (
	flagProxy    = flag.String("x", "", "proxy")
	flagChangeCL = flag.Bool("cl", false, "recalculate content length")
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
		fmt.Printf("flags: flagProxy(%q), flagChangeCL(%t), flagDebug(%t)\n", *flagProxy, *flagChangeCL, *flagDebug)
		fmt.Printf("Data: %q\n", data)
	}

	if isH2Msg(data) {
		doH2Request(data)
		return
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
	client.Timeout = 30 * time.Second
	client.ReadFull = true
	client.QuietTimeout = 500 * time.Millisecond

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

	if input.URL == "" {
		return nil, fmt.Errorf("invalid format: empty target URL")
	}

	req := parts[1]
	// Check for header/body delimiter in both LF and CRLF forms
	hasDelimiter := bytes.Contains(req, []byte("\n\n")) || bytes.Contains(req, []byte("\r\n\r\n"))
	if !hasDelimiter {
		req = bytes.TrimSpace(req)
		req = append(req, []byte("\n\n")...)
	}
	input.Request = dataNormalization(req)

	return input, nil
}

func dataNormalization(req []byte) []byte {
	req = replaceLastCR(req)
	// Collapse \r\n to \n first, then replace any remaining bare \r
	// (old Mac-style line endings) with \n, then convert all \n to \r\n.
	req = bytes.ReplaceAll(req, []byte("\r\n"), []byte("\n"))
	req = bytes.ReplaceAll(req, []byte("\r"), []byte("\n"))
	req = bytes.ReplaceAll(req, []byte("\n"), []byte("\r\n"))

	// If a body exists after the header delimiter, strip a single
	// trailing \r\n that comes from Unix/Windows text file line endings.
	delim := []byte("\r\n\r\n")
	idx := bytes.Index(req, delim)
	if idx >= 0 {
		body := req[idx+len(delim):]
		if len(body) > 0 && bytes.HasSuffix(body, []byte("\r\n")) {
			req = req[:len(req)-2]
		}
	}

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

func isH2Msg(data []byte) bool {
	s := string(data)
	if strings.HasPrefix(s, rawhttp2.H2MsgMagic) {
		return true
	}
	if strings.HasPrefix(s, "#") {
		idx := strings.Index(s, "\n")
		if idx >= 0 && strings.HasPrefix(strings.TrimSpace(s[idx+1:]), rawhttp2.H2MsgMagic) {
			return true
		}
	}
	return false
}

func doH2Request(data []byte) {
	s := string(data)

	target := ""
	h2data := data

	if strings.HasPrefix(s, "#") && !strings.HasPrefix(s, rawhttp2.H2MsgMagic) {
		idx := strings.Index(s, "\n")
		if idx >= 0 {
			target = strings.TrimSpace(s[1:idx])
			h2data = []byte(s[idx+1:])
		}
	}

	msg, err := rawhttp2.ParseH2Msg(h2data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "h2msg parse error: %v\n", err)
		os.Exit(1)
	}

	if target == "" {
		for _, h := range msg.Headers {
			switch string(h.NameDecoded) {
			case ":authority":
				target = "https://" + string(h.ValueDecoded)
			case ":scheme":
				if target == "" {
					target = string(h.ValueDecoded) + "://"
				}
			}
		}
	}

	if target == "" {
		fmt.Fprintf(os.Stderr, "h2 target not found: provide #URL line or :authority pseudoheader\n")
		os.Exit(1)
	}

	if !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") {
		target = "https://" + target
	}

	client := rawhttp2.NewDefaultClient()
	client.Timeout = 30 * time.Second

	respMsg, h2err, err := client.Do(target, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "h2 request error: %v\n", err)
		os.Exit(1)
	}

	if h2err != nil {
		fmt.Fprintf(os.Stderr, "h2 protocol error: %s (code=%d, scope=%s)\n", h2err.Message, h2err.Code, h2err.Scope)
		if respMsg == nil {
			os.Exit(1)
		}
	}

	fmt.Printf("%s", rawhttp2.SerializeH2Msg(respMsg))
}
