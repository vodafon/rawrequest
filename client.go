package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"
)

var (
	InvalidURLError     = fmt.Errorf("Invalid URL")
	InvalidRequestError = fmt.Errorf("Invalid Request")
)

type ReadWriteCloseDeadliner interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
}

type Client struct {
	TransformRequestFunc func(*Request)
	Timeout              time.Duration
}

type Response struct {
	Rawdata []byte
}

type Request struct {
	Rawdata []byte
	Addr    string
	IsTLS   bool
}

func (obj *Client) NewResponse() *Response {
	return &Response{}
}

func (obj *Client) NewRequest() *Request {
	return &Request{}
}

func (obj *Client) NewRequestResponse() (*Request, *Response) {
	return obj.NewRequest(), obj.NewResponse()
}

func NewDefaultClient() *Client {
	return &Client{
		TransformRequestFunc: PrepareRequest,
		Timeout:              time.Second * 10,
	}
}

func NewDefaultClientTimeout(d time.Duration) *Client {
	return &Client{
		TransformRequestFunc: PrepareRequest,
		Timeout:              d,
	}
}

func (obj *Client) Do(req *Request, resp *Response) error {
	obj.TransformRequestFunc(req)
	if req.IsTLS {
		return obj.DoHTTPS(req, resp)
	}
	return obj.DoHTTP(req, resp)
}

func (obj *Client) DoHTTPS(req *Request, resp *Response) error {
	dialer := &net.Dialer{Timeout: obj.Timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", req.Addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return err
	}
	return obj.DoConn(conn, req, resp)
}

func (obj *Client) DoHTTP(req *Request, resp *Response) error {
	conn, err := net.DialTimeout("tcp", req.Addr, obj.Timeout)
	if err != nil {
		return err
	}
	return obj.DoConn(conn, req, resp)
}

func (obj *Client) DoConn(conn ReadWriteCloseDeadliner, req *Request, resp *Response) error {
	defer conn.Close()
	conn.Write(req.Rawdata)
	bufReader := bufio.NewReader(conn)

	for {
		// Set a deadline for reading. Read operation will fail if no data
		// is received after deadline.
		conn.SetReadDeadline(time.Now().Add(obj.Timeout))

		// Read tokens delimited by newline
		bytes, err := bufReader.ReadBytes('\n')
		resp.Rawdata = append(resp.Rawdata, bytes...)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func PrepareRequest(req *Request) {
	req.Rawdata = bytes.ReplaceAll(req.Rawdata, []byte("\r\n"), []byte("\n"))
	req.Rawdata = bytes.ReplaceAll(req.Rawdata, []byte("\n"), []byte("\r\n"))
	req.Rawdata = bytes.ReplaceAll(req.Rawdata, []byte("||CR||"), []byte("\r"))
	req.Rawdata = bytes.ReplaceAll(req.Rawdata, []byte("||LF||"), []byte("\n"))
	if bytes.Contains(req.Rawdata, []byte("||CLEN||")) {
		ContentLengthCalculation(req)
	}
}

func ContentLengthCalculation(req *Request) {
	parts := bytes.Split(bytes.TrimSpace(req.Rawdata), []byte("\r\n\r\n"))
	if len(parts) < 2 {
		req.Rawdata = bytes.ReplaceAll(req.Rawdata, []byte("||CLEN||"), []byte("0"))
		return
	}
	l := len(bytes.Join(parts[1:], []byte("\r\n\r\n")))
	req.Rawdata = bytes.ReplaceAll(req.Rawdata, []byte("||CLEN||"), []byte(fmt.Sprintf("%d", l)))
}
