package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type httpRequest struct {
	Method    string
	Url       string
	PathParam string
}

func main() {
	log.Printf("Listening on 4221...")
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	defer conn.Close()

	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	var response string
	httpReq := requestHandler(conn)

	switch httpReq.Url {
	case "":
		response = "HTTP/1.1 200 OK\r\n\r\n"
	case "/":
		response = "HTTP/1.1 200 OK\r\n\r\n"
	case "echo":
		log.Printf("path params -> %v", httpReq.PathParam)
		length := len(httpReq.PathParam)
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %v\r\n\r\n%v", length, httpReq.PathParam)
	default:
		log.Printf("request -> %v , path param -> %v", httpReq.Url, httpReq.PathParam)
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing to connection")
		os.Exit(1)
	}
}

func requestHandler(conn net.Conn) httpRequest {
	reader := bufio.NewReader(conn)
	buffer := make([]byte, 0, 4096)
	temp := make([]byte, 512)

	for {
		n, err := reader.Read(temp)

		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading:", err.Error())
			}
			break
		}
		buffer = append(buffer, temp[:n]...)
		if strings.Contains(string(buffer), "\r\n\r\n") {
			break
		}

	}

	request := string(buffer)
	log.Printf("Request: %v", request)
	header := strings.SplitN(request, "\r\n", 2)[0]
	headerParts := strings.Split(header, " ")
	pathParams := strings.Split(headerParts[1], "/")

	var pathParam string
	log.Printf("path params len %v", len(pathParams))
	if len(pathParams) > 2 {
		pathParam = strings.TrimSpace(pathParams[2])
	}
	url := pathParams[1]
	return httpRequest{
		Method:    headerParts[0],
		Url:       url,
		PathParam: pathParam,
	}
}
