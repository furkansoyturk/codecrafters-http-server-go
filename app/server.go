package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

type Headers map[string]string

const (
	HTTP_OK        = "HTTP/1.1 200 OK"
	HTTP_CREATED   = "HTTP/1.1 201 Created"
	HTTP_NOT_FOUND = "HTTP/1.1 404 Not Found"
)

type Request struct {
	HttpVerb string
	Path     string
	Protocol string
	Headers  Headers
	Body     string
}

func ParseRequest(rawRequest string) Request {
	reqLine, rawRemainder, found := strings.Cut(rawRequest, "\r\n")
	if !found {
		fmt.Println("Request malformed")
		os.Exit(1)
	}
	reqLineParts := strings.Split(reqLine, " ")
	blobbedHeaders, body, found := strings.Cut(rawRemainder, "\r\n\r\n")
	if !found {
		fmt.Println("Request malformed")
		os.Exit(1)
	}
	rawHeaders := strings.Split(blobbedHeaders, "\r\n")
	headerMap := make(Headers)
	for _, rawHeader := range rawHeaders {
		key, val, found := strings.Cut(rawHeader, ":")
		if !found {
			fmt.Println("Header malformed")
			os.Exit(1)
		}
		headerMap[key] = strings.TrimSpace(val)
	}
	return Request{
		HttpVerb: reqLineParts[0],
		Path:     reqLineParts[1],
		Protocol: reqLineParts[2],
		Headers:  headerMap,
		Body:     body,
	}
}
func WriteResponse(conn net.Conn, status string, headers Headers, body string) {
	conn.Write([]byte(status))
	conn.Write([]byte("\r\n"))
	if headers != nil {
		for headerKey, headerVal := range headers {
			hString := fmt.Sprintf("%s: %s\r\n", headerKey, headerVal)
			conn.Write([]byte(hString))
		}
	}
	conn.Write([]byte("\r\n"))
	if body != "" {
		conn.Write([]byte(body))
	}
}
func GetEchoString(req Request) string {
	_, echoStr, _ := strings.Cut(req.Path[1:], "/")
	return echoStr
}
func HandleGetFileCall(conn net.Conn, req Request, dir string) {
	pathParts := strings.Split(req.Path, "/")
	filename := pathParts[len(pathParts)-1]
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		WriteResponse(conn, HTTP_NOT_FOUND, nil, "")
	}
	file, err := os.Open(filepath)
	if err != nil {
		WriteResponse(conn, HTTP_NOT_FOUND, nil, "")
	}
	fileSize := fileInfo.Size()
	contents := make([]byte, fileSize)
	bytesRead, err := file.Read(contents)
	if err != nil {
		WriteResponse(conn, HTTP_NOT_FOUND, nil, "")
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/octet-stream"
	headers["Content-Length"] = fmt.Sprintf("%d", bytesRead)
	WriteResponse(conn, HTTP_OK, headers, string(contents))
}
func HandlePostFileCall(conn net.Conn, req Request, dir string) {
	pathParts := strings.Split(req.Path, "/")
	filename := pathParts[len(pathParts)-1]
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	rawOut := []byte(req.Body)
	out := bytes.Trim(rawOut, "\x00")
	err := os.WriteFile(filepath, out, 0644)
	if err != nil {
		WriteResponse(conn, "500", nil, "")
	}
	headers := make(Headers)
	headers["Content-Length"] = "0"
	WriteResponse(conn, HTTP_CREATED, headers, "")
}
func HandleEchoCall(conn net.Conn, req Request) {
	content := GetEchoString(req)
	returnHeaders := make(Headers)
	returnHeaders["Content-Type"] = "text/plain"
	returnHeaders["Content-Length"] = fmt.Sprintf("%d", (len(content)))
	rawEncodings, ok := req.Headers["Accept-Encoding"]
	if ok {
		encodings := strings.Split(rawEncodings, ",")
		for _, rawEncoding := range encodings {
			encoding := strings.TrimSpace(rawEncoding)
			if encoding[len(encoding)-1] == []byte(",")[0] {
				encoding = encoding[:len(encoding)-2]
			}
			if encoding == "gzip" {
				returnHeaders["Content-Encoding"] = encoding
				var compressedContent bytes.Buffer
				w := gzip.NewWriter(&compressedContent)
				w.Write([]byte(content))
				w.Close()
				returnHeaders["Content-Length"] = fmt.Sprintf("%d", compressedContent.Len())
				WriteResponse(conn, HTTP_OK, returnHeaders, string(compressedContent.Bytes()))
				return
			}
		}
	}
	WriteResponse(conn, HTTP_OK, returnHeaders, content)
}
func shouldEcho(req Request) bool {
	verbOk := req.HttpVerb == "GET"
	pathOk, _ := regexp.Match("/echo/.+", []byte(req.Path))
	return verbOk && pathOk
}
func shouldGetFile(req Request) bool {
	verbOk := req.HttpVerb == "GET"
	pathOk, _ := regexp.Match("/files/.+", []byte(req.Path))
	return verbOk && pathOk
}
func shouldPostFile(req Request) bool {
	verbOk := req.HttpVerb == "POST"
	pathOk, _ := regexp.Match("/files/.+", []byte(req.Path))
	return verbOk && pathOk
}
func HandleRequest(conn net.Conn, dir string) {
	bData := make([]byte, 1024)
	conn.Read(bData)
	data := string(bData)
	req := ParseRequest(data)
	if shouldGetFile(req) {
		HandleGetFileCall(conn, req, dir)
		return
	}
	if shouldPostFile(req) {
		HandlePostFileCall(conn, req, dir)
		return
	}
	if shouldEcho(req) {
		HandleEchoCall(conn, req)
		return
	}
	if agent, ok := req.Headers["User-Agent"]; ok {
		headers := make(Headers)
		headers["Content-Type"] = "text/plain"
		headers["Content-Length"] = fmt.Sprintf("%d", len(agent))
		WriteResponse(conn, HTTP_OK, headers, agent)
		return
	}
	if strings.Contains(data, "GET / HTTP/1.1") {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	}
	conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
}
func ParseArgs(args []string) string {
	dirIdx := -1
	for i, arg := range args {
		if arg == "--directory" {
			dirIdx = i + 1
		}
	}
	if dirIdx == -1 {
		return ""
	}
	return args[dirIdx]
}
func main() {
	dir := ParseArgs(os.Args[1:])
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	newConns := make(chan net.Conn)
	go func(l net.Listener) {
		for {
			c, err := l.Accept()
			if err != nil {
				newConns <- nil
				return
			}
			newConns <- c
		}
	}(l)
	for {
		select {
		case c := <-newConns:
			HandleRequest(c, dir)
		}
	}
}
