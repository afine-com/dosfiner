# dosfiner

**dosfiner** is a simple concurrency-based HTTP stress/testing tool, supporting GET/POST requests, custom headers, and proxy usage. It can also handle raw requests (`-r`) similarly to sqlmap’s `-r` approach, and has a `--force-ssl` option.

## Features

- Concurrency: Launch multiple goroutines (`-t`) for GET or POST floods
- Custom headers (`-H`) or data (`-d`)
- Proxy support (`-proxy`)  
- Raw request mode (`-r file.txt`)
- Force-SSL option (`--force-ssl`), switching `http://` → `https://`

## Installation

```bash
git clone https://github.com/afine-com/dosfiner.git
cd dosfiner
go build dosfiner.go
# or simply:
go run dosfiner.go [options]
