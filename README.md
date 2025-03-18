
# Dosfiner – HTTP Stress & DoS Testing Tool 🚀

**Dosfiner** is a lightweight and efficient HTTP stress-testing and Denial-of-Service (DoS) tool written in Go. Quickly simulate high-volume HTTP traffic to identify application performance bottlenecks and vulnerabilities.

## ⚡ Main Features
- Perform rapid GET/POST request flooding.
- Custom concurrency level using threads.
- Send HTTP requests directly from RAW files.
- Supports HTTP proxy (e.g., Burp Suite).

## 📥 Installation
1. Clone the repository:
```bash
git clone https://github.com/afine-com/dosfiner.git
cd dosfiner
```
2. Build or Run Directly:
- Build binary:
```bash
go build dosfiner.go
```
- Run directly:
```bash
go run dosfiner.go [options]
```

## 🚀 Usage Examples
- **GET request flooding (500 threads):**
```bash
./dosfiner -g -u "https://target.com/api/v1/search" -t 500
```
- **POST request flooding with data (300 threads):**
```bash
./dosfiner -p -u "https://target.com/login" -d "user=admin&pass=test" -t 300
```
- **Using HTTP Proxy (Burp Suite):**
```bash
./dosfiner -g -u "https://target.com/api" -proxy "http://127.0.0.1:8080" -t 200
```
- **RAW HTTP Request from File:**
```bash
./dosfiner -r "/tmp/request.txt" -t 400 --force-ssl
```

## ⚠️ Important Notice
**Use responsibly.**  
Dosfiner generates intense traffic that may lead to real Denial-of-Service scenarios. Always obtain explicit permission and perform testing in controlled environments only.

## 📜 License
[MIT License](LICENSE)
