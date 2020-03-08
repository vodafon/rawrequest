package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
)

var (
	flagFile = flag.String("file", "", "path to request file")
	flagAddr = flag.String("addr", "", "example.com:80")
	flagTLS  = flag.Bool("tls", false, "TLS connect")
)

func main() {
	flag.Parse()
	if *flagFile == "" || *flagAddr == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	client := NewDefaultClient()
	req, resp := client.NewRequestResponse()
	req.Addr = *flagAddr
	req.IsTLS = *flagTLS
	req.Rawdata = loadRequest(*flagFile)
	err := client.Do(req, resp)
	if err != nil {
		log.Fatalf("Do request error: %v", err)
	}
}

func loadRequest(filename string) []byte {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Load request error: %v", err)
	}
	return file
}
