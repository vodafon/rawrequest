package main

import (
	"bytes"
	"testing"
)

// --- replaceLastCR ---

func TestReplaceLastCR(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "empty input",
			in:   []byte{},
			want: []byte{},
		},
		{
			name: "ends with LF unchanged",
			in:   []byte("hello\n"),
			want: []byte("hello\n"),
		},
		{
			name: "ends with CR replaced",
			in:   []byte("hello\r"),
			want: []byte("hello\n"),
		},
		{
			name: "ends with CRLF unchanged",
			in:   []byte("hello\r\n"),
			want: []byte("hello\r\n"),
		},
		{
			name: "single CR",
			in:   []byte{'\r'},
			want: []byte{'\n'},
		},
		{
			name: "no trailing newline unchanged",
			in:   []byte("hello"),
			want: []byte("hello"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceLastCR(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("replaceLastCR(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestReplaceLastCR_DoesNotMutateInput(t *testing.T) {
	orig := []byte("hello\r")
	input := make([]byte, len(orig))
	copy(input, orig)

	_ = replaceLastCR(input)

	if !bytes.Equal(input, orig) {
		t.Errorf("replaceLastCR mutated input: got %q, want %q", input, orig)
	}
}

// --- dataNormalization ---

func TestDataNormalization(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "LF only to CRLF",
			in:   []byte("GET / HTTP/1.1\nHost: example.com\n\n"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name: "already CRLF unchanged",
			in:   []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name: "mixed LF and CRLF normalized",
			in:   []byte("GET / HTTP/1.1\r\nHost: example.com\nConnection: close\r\n\r\n"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n"),
		},
		{
			name: "trailing CR converted",
			in:   []byte("GET / HTTP/1.1\nHost: example.com\n\r"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name: "empty input",
			in:   []byte{},
			want: []byte{},
		},
		// Bare \r (old Mac-style) is now normalized to \r\n.
		{
			name: "bare CR normalized to CRLF",
			in:   []byte("GET / HTTP/1.1\rHost: example.com\r\n\r\n"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataNormalization(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("dataNormalization(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- getHostRegex ---

func TestGetHostRegex(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{
			name: "standard host header",
			in:   []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			want: "example.com",
		},
		{
			name: "host with port",
			in:   []byte("GET / HTTP/1.1\r\nHost: example.com:8080\r\n\r\n"),
			want: "example.com:8080",
		},
		{
			name: "case insensitive host",
			in:   []byte("GET / HTTP/1.1\r\nHOST: example.com\r\n\r\n"),
			want: "example.com",
		},
		{
			name: "lowercase host",
			in:   []byte("GET / HTTP/1.1\r\nhost: example.com\r\n\r\n"),
			want: "example.com",
		},
		{
			name: "mixed case host",
			in:   []byte("GET / HTTP/1.1\r\nHoSt: example.com\r\n\r\n"),
			want: "example.com",
		},
		{
			name: "host with trailing CR (LF line endings)",
			in:   []byte("GET / HTTP/1.1\nHost: example.com\n\n"),
			want: "example.com",
		},
		{
			name: "host with extra whitespace after colon",
			in:   []byte("GET / HTTP/1.1\r\nHost:   example.com\r\n\r\n"),
			want: "example.com",
		},
		{
			name: "host with no whitespace after colon",
			in:   []byte("GET / HTTP/1.1\r\nHost:example.com\r\n\r\n"),
			want: "example.com",
		},
		{
			name: "no host header returns empty",
			in:   []byte("GET / HTTP/1.1\r\nConnection: close\r\n\r\n"),
			want: "",
		},
		{
			name: "empty input",
			in:   []byte{},
			want: "",
		},
		{
			name: "multiple host headers returns first",
			in:   []byte("GET / HTTP/1.1\r\nHost: first.com\r\nHost: second.com\r\n\r\n"),
			want: "first.com",
		},
		{
			name: "host header not at start of line ignored",
			in:   []byte("GET / HTTP/1.1\r\nX-Host: fake.com\r\nHost: real.com\r\n\r\n"),
			want: "real.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getHostRegex(tt.in)
			if got != tt.want {
				t.Errorf("getHostRegex(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- replaceContentLength ---

func TestReplaceContentLength(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "standard content-length replaced",
			in:   []byte("POST / HTTP/1.1\r\nContent-Length: 42\r\nHost: example.com\r\n\r\nbody"),
			want: []byte("POST / HTTP/1.1\r\nContent-Length: ||CLEN||\r\nHost: example.com\r\n\r\nbody"),
		},
		{
			name: "case insensitive replacement",
			in:   []byte("content-length: 100\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\n"),
		},
		{
			name: "mixed case replacement",
			in:   []byte("CONTENT-LENGTH: 100\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\n"),
		},
		{
			name: "content-length zero",
			in:   []byte("Content-Length: 0\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\n"),
		},
		{
			name: "content-length large number",
			in:   []byte("Content-Length: 999999999\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\n"),
		},
		{
			name: "no content-length unchanged",
			in:   []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name: "multiple content-length headers both replaced",
			in:   []byte("Content-Length: 10\r\nHost: x\r\nContent-Length: 20\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\nHost: x\r\nContent-Length: ||CLEN||\r\n"),
		},
		{
			name: "content-length no space after colon",
			in:   []byte("Content-Length:42\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\n"),
		},
		{
			name: "content-length multiple spaces",
			in:   []byte("Content-Length:   42\r\n"),
			want: []byte("Content-Length: ||CLEN||\r\n"),
		},
		{
			name: "empty content-length value not replaced (known limitation)",
			// Content-Length with no digit → regex requires \d+ → no match
			in:   []byte("Content-Length: \r\n"),
			want: []byte("Content-Length: \r\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceContentLength(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("replaceContentLength(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- ParseData ---

func TestParseData(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		wantURL string
		wantReq []byte
		wantErr bool
	}{
		{
			name:    "with target line dispatches to ParseDataWithTarget",
			in:      []byte("#https://example.com\nGET / HTTP/1.1\nHost: example.com\n\n"),
			wantURL: "https://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "without target line extracts host",
			in:      []byte("GET / HTTP/1.1\nHost: example.com\n\n"),
			wantURL: "https://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "without target and host with port",
			in:      []byte("GET / HTTP/1.1\nHost: example.com:8080\n\n"),
			wantURL: "https://example.com:8080",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com:8080\r\n\r\n"),
		},
		{
			name:    "no target and no host header errors",
			in:      []byte("GET / HTTP/1.1\nConnection: close\n\n"),
			wantErr: true,
		},
		{
			name:    "empty input errors",
			in:      []byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseData(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseData(%q) expected error, got nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseData(%q) unexpected error: %v", tt.in, err)
			}
			if got.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tt.wantURL)
			}
			if !bytes.Equal(got.Request, tt.wantReq) {
				t.Errorf("Request = %q, want %q", got.Request, tt.wantReq)
			}
		})
	}
}

// --- ParseDataWithTarget ---

func TestParseDataWithTarget(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		wantURL string
		wantReq []byte
		wantErr bool
	}{
		{
			name:    "standard format with LF",
			in:      []byte("#https://example.com\nGET / HTTP/1.1\nHost: example.com\n\n"),
			wantURL: "https://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "URL with whitespace trimmed",
			in:      []byte("#  https://example.com  \nGET / HTTP/1.1\nHost: example.com\n\n"),
			wantURL: "https://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "no newline after URL errors",
			in:      []byte("#https://example.com"),
			wantErr: true,
		},
		{
			name:    "request without double newline gets one appended",
			in:      []byte("#https://example.com\nGET / HTTP/1.1\nHost: example.com"),
			wantURL: "https://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "request with CRLF line endings",
			in:      []byte("#https://example.com\nGET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			wantURL: "https://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "http URL preserved",
			in:      []byte("#http://example.com\nGET / HTTP/1.1\nHost: example.com\n\n"),
			wantURL: "http://example.com",
			wantReq: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "URL with path and query",
			in:      []byte("#https://example.com/path?q=1\nGET /path?q=1 HTTP/1.1\nHost: example.com\n\n"),
			wantURL: "https://example.com/path?q=1",
			wantReq: []byte("GET /path?q=1 HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name:    "POST with body",
			in:      []byte("#https://example.com\nPOST / HTTP/1.1\nHost: example.com\nContent-Length: 4\n\nbody"),
			wantURL: "https://example.com",
			wantReq: []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\nbody"),
		},
		{
			name:    "empty URL returns error",
			in:      []byte("#\nGET / HTTP/1.1\nHost: example.com\n\n"),
			wantErr: true,
		},
		{
			name:    "whitespace only URL returns error",
			in:      []byte("#   \nGET / HTTP/1.1\nHost: example.com\n\n"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDataWithTarget(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseDataWithTarget(%q) expected error, got nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDataWithTarget(%q) unexpected error: %v", tt.in, err)
			}
			if got.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tt.wantURL)
			}
			if !bytes.Equal(got.Request, tt.wantReq) {
				t.Errorf("Request = %q, want %q", got.Request, tt.wantReq)
			}
		})
	}
}

// Verify CRLF delimiter is properly detected so body is preserved.
func TestParseDataWithTarget_CRLFDelimiter(t *testing.T) {
	input := []byte("#https://example.com\nPOST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\nbody")

	got, err := ParseDataWithTarget(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\nbody")
	if !bytes.Equal(got.Request, want) {
		t.Errorf("CRLF delimiter not handled correctly\ngot  = %q\nwant = %q", got.Request, want)
	}
}

// Verify bare \r is normalized to \r\n.
func TestDataNormalization_BareCR(t *testing.T) {
	input := []byte("GET / HTTP/1.1\rHost: example.com\r\n\r\n")

	got := dataNormalization(input)
	want := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")

	if !bytes.Equal(got, want) {
		t.Errorf("bare \\r not converted to \\r\\n\ngot  = %q\nwant = %q", got, want)
	}
}

// Verify trailing \n from Unix text files is stripped from body.
// Unix files end with \n which is not part of the HTTP body.
func TestDataNormalization_TrailingNewlineStrippedFromBody(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "body with trailing LF",
			in:   []byte("POST / HTTP/1.1\nHost: example.com\nContent-Length: 3\n\na=1\n"),
			want: []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 3\r\n\r\na=1"),
		},
		{
			name: "body with trailing CRLF",
			in:   []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 3\r\n\r\na=1\r\n"),
			want: []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 3\r\n\r\na=1"),
		},
		{
			name: "body without trailing newline unchanged",
			in:   []byte("POST / HTTP/1.1\nHost: example.com\nContent-Length: 3\n\na=1"),
			want: []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 3\r\n\r\na=1"),
		},
		{
			name: "no body unchanged",
			in:   []byte("GET / HTTP/1.1\nHost: example.com\n\n"),
			want: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		},
		{
			name: "form body with trailing LF",
			in:   []byte("POST / HTTP/1.1\nHost: example.com\nContent-Type: application/x-www-form-urlencoded\nContent-Length: 11\n\nuser=a&x=42\n"),
			want: []byte("POST / HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: 11\r\n\r\nuser=a&x=42"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataNormalization(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("dataNormalization(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
