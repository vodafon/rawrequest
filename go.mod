module github.com/vodafon/rawrequest

go 1.25.5

require github.com/vodafon/rawhttp v0.3.1

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/vodafon/rawhttp2 v0.0.0
	golang.org/x/net v0.52.0 // indirect
)

replace github.com/vodafon/rawhttp2 => ../rawhttp2
