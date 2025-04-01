package main

import (
    "context"
    "flag"
    "fmt"
    "math/rand"
    "net"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"
)

// ---------- Globals / Flags ----------

var (
    targetURL       string
    dataPayload     string
    concurrency     int
    proxyAddress    string
    httpHeaders     headerSlice
    totalRequests   int
    printedMessages []string
    wg              sync.WaitGroup
    flagGet         bool
    flagPost        bool

    rawRequestFile string
    forceSSL       bool
)

type headerSlice []string

func (h *headerSlice) String() string { return fmt.Sprintf("%v", *h) }
func (h *headerSlice) Set(value string) error {
    *h = append(*h, value)
    return nil
}

func forceCRLF(input string) string {
    placeholder := "\x00CRLF\x00"
    out := strings.ReplaceAll(input, "\r\n", placeholder)
    out = strings.ReplaceAll(out, "\n", "\r\n")
    out = strings.ReplaceAll(out, placeholder, "\r\n")
    return out
}

// -------------------------------------

func main() {
    flag.BoolVar(&flagGet, "g", false, "Use GET request (requires -u)")
    flag.BoolVar(&flagPost, "p", false, "Use POST request (requires -u)")
    flag.StringVar(&targetURL, "u", "", "Target URL (ignored if -r used)")
    flag.StringVar(&dataPayload, "d", "", "POST data (x-www-form-urlencoded)")
    flag.IntVar(&concurrency, "t", 500, "Number of concurrent threads")
    flag.StringVar(&proxyAddress, "proxy", "", "Proxy address (http://127.0.0.1:8080)")
    flag.Var(&httpHeaders, "H", "Custom HTTP header (repeatable)")

    flag.StringVar(&rawRequestFile, "r", "", "Read raw HTTP request from file (like sqlmap -r)")
    flag.BoolVar(&forceSSL, "force-ssl", false, "Force https")

		flag.Usage = func() {
		    fmt.Println("Usage: dosfiner [options]")
		    fmt.Println("  -g, -p, -u, -d, -H, -t, -proxy, etc.")
		    fmt.Println("  -r <file> to send raw request from file")
		    fmt.Println("  --force-ssl to switch http -> https")
		    fmt.Println("Example: dosfiner -r request.txt -t 10 --force-ssl -proxy http://127.0.0.1:8080")
		}

    rand.Seed(time.Now().UnixNano())
    flag.Parse()

    if rawRequestFile != "" {
        // Raw mode
        rawData, err := parseRawRequestFromFile(rawRequestFile)
        if err != nil {
            fmt.Println("Error parsing raw request:", err)
            return
        }
        client := createHTTPClient(proxyAddress)
        wg.Add(concurrency)
        for i := 0; i < concurrency; i++ {
            go doRawRequest(client, rawData)
        }
        wg.Wait()
        fmt.Println("\nFinished sending requests (raw mode).")
        return
    }

    // Normal mode
    if targetURL == "" {
        fmt.Println("You must specify -u or use -r.")
        flag.Usage()
        return
    }
    if !flagGet && !flagPost {
        fmt.Println("You must choose -g or -p (unless using -r).")
        flag.Usage()
        return
    }

    if forceSSL {
        if strings.HasPrefix(strings.ToLower(targetURL), "http://") {
            targetURL = "https://" + targetURL[7:]
        } else if !strings.HasPrefix(strings.ToLower(targetURL), "https://") {
            targetURL = "https://" + targetURL
        }
    }

    client := createHTTPClient(proxyAddress)

    // Parse -H flags
    headerMap := make(map[string]string)
    for _, h := range httpHeaders {
        parts := strings.SplitN(h, ":", 2)
        if len(parts) == 2 {
            name := strings.TrimSpace(parts[0])
            value := strings.TrimSpace(parts[1])
            headerMap[name] = value
        }
    }

    wg.Add(concurrency)
    for i := 0; i < concurrency; i++ {
        if flagPost {
            go doPOST(client, headerMap, targetURL, dataPayload)
        } else {
            go doGET(client, headerMap, targetURL)
        }
    }
    wg.Wait()
    fmt.Println("\nFinished sending requests.")
}

// -------------- Creating client with minimal rewriting --------------
func createHTTPClient(proxyAddr string) *http.Client {
    // custom transport to reduce rewriting
    tr := &http.Transport{
        DisableCompression: true,    // no transparent gz
        ForceAttemptHTTP2:  false,   // skip http2
        DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
            d := net.Dialer{}
            return d.DialContext(ctx, network, address)
        },
    }
    if proxyAddr != "" {
        proxyURL, err := url.Parse(proxyAddr)
        if err == nil {
            tr.Proxy = http.ProxyURL(proxyURL)
        }
    }

    return &http.Client{
        Transport: tr,
        Timeout:   30 * time.Second,
    }
}

// -------------- Normal GET/POST --------------
func doGET(client *http.Client, headers map[string]string, urlStr string) {
    defer wg.Done()
    req, err := http.NewRequest("GET", urlStr, nil)
    if err != nil {
        return
    }
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    resp, err := client.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()
    handleResponseCode(resp.StatusCode)
}

func doPOST(client *http.Client, headers map[string]string, urlStr, bodyStr string) {
    defer wg.Done()
    req, err := http.NewRequest("POST", urlStr, strings.NewReader(bodyStr))
    if err != nil {
        return
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    resp, err := client.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()
    handleResponseCode(resp.StatusCode)
}

// -------------- RAW MODE --------------
type rawRequestData struct {
    Method  string
    URL     string
    Headers map[string]string
    Body    string
}

func doRawRequest(client *http.Client, raw *rawRequestData) {
    defer wg.Done()
    bodyReader := strings.NewReader(raw.Body)

    req, err := http.NewRequest(raw.Method, raw.URL, bodyReader)
    if err != nil {
        return
    }
    req.ContentLength = int64(len(raw.Body))

    for k, v := range raw.Headers {
        if strings.ToLower(k) == "content-length" {
            continue
        }
        req.Header.Set(k, v)
    }

    resp, err := client.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()
    handleResponseCode(resp.StatusCode)
}

func parseRawRequestFromFile(filePath string) (*rawRequestData, error) {
    rawBytes, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    rawStr := string(rawBytes)

    sepIndex := strings.Index(rawStr, "\r\n\r\n")
    sepLen := 4
    if sepIndex == -1 {
        sepIndex = strings.Index(rawStr, "\n\n")
        sepLen = 2
    }

    headerPart := rawStr
    bodyPart := ""
    if sepIndex != -1 {
        headerPart = rawStr[:sepIndex]
        bodyPart = rawStr[sepIndex+sepLen:]
    }

    var lines []string
    if strings.Contains(headerPart, "\r\n") {
        lines = strings.Split(headerPart, "\r\n")
    } else {
        lines = strings.Split(headerPart, "\n")
    }

    if len(lines) < 1 {
        return nil, fmt.Errorf("invalid request file (no lines)")
    }

    firstLine := lines[0]
    parts := strings.SplitN(firstLine, " ", 3)
    if len(parts) < 2 {
        return nil, fmt.Errorf("invalid request line: %s", firstLine)
    }
    method := parts[0]
    path := parts[1]

    hdrMap := make(map[string]string)
    for i := 1; i < len(lines); i++ {
        line := strings.TrimSpace(lines[i])
        if line == "" {
            continue
        }
        kv := strings.SplitN(line, ":", 2)
        if len(kv) == 2 {
            k := strings.TrimSpace(kv[0])
            v := strings.TrimSpace(kv[1])
            hdrMap[k] = v
        }
    }
    host := hdrMap["Host"]
    if host == "" {
        return nil, fmt.Errorf("no Host header found")
    }

    scheme := "http"
    if strings.Contains(host, ":443") {
        scheme = "https"
    }
    if forceSSL {
        scheme = "https"
    }

    if ct, ok := hdrMap["Content-Type"]; ok {
        if strings.Contains(strings.ToLower(ct), "multipart/form-data") {
            bodyPart = forceCRLF(bodyPart)
        }
    }

    fullURL := fmt.Sprintf("%s://%s%s", scheme, host, path)
    return &rawRequestData{
        Method:  method,
        URL:     fullURL,
        Headers: hdrMap,
        Body:    bodyPart,
    }, nil
}

// -------------- Helpers --------------
func handleResponseCode(sc int) {
    totalRequests++
    fmt.Printf("\r%d requests have been sent", totalRequests)
    if sc == 429 {
        printOnce("You have been throttled (429)")
    } else if sc == 500 {
        printOnce("Status code 500 received")
    }
}

func printOnce(msg string) {
    if !strings.Contains(strings.Join(printedMessages, " "), msg) {
        fmt.Printf("\n%s after %d requests\n", msg, totalRequests)
        printedMessages = append(printedMessages, msg)
    }
}
