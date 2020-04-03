package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var (
	flagFile               = flag.String("file", "", "path to request file")
	flagAddr               = flag.String("addr", "", "example.com:80")
	flagTLS                = flag.Bool("tls", false, "TLS connect")
	flagTimes              = flag.Int("times", 1, "repeat requests")
	flagPrintResponseMatch = flag.String("response-match", "", "print responses if match")
)

func main() {
	flag.Parse()
	if *flagFile == "" || *flagAddr == "" || *flagTimes < 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	bytesMatch := []byte(*flagPrintResponseMatch)
	lMatch := len(bytesMatch)
	client := NewDefaultClient()
	for i := 0; i < *flagTimes; i++ {
		req, resp := client.NewRequestResponse()
		req.Addr = *flagAddr
		req.IsTLS = *flagTLS
		req.Rawdata = loadRequest(*flagFile)
		err := client.Do(req, resp)
		if err != nil {
			log.Fatalf("Do request error: %v", err)
		}
		if lMatch > 0 && bytes.Contains(resp.Rawdata, bytesMatch) {
			fmt.Printf("\n\n%s\n\n", resp.Rawdata)
		}
	}
}

func loadRequest(filename string) []byte {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Load request error: %v", err)
	}
	return file
}
