package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type httpRequest struct {
	Method string
	Url    string
}

func main() {
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

	if httpReq.Url == "/" {
		response = "HTTP/1.1 200 OK\r\n\r\n"
	} else {
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
	header := strings.SplitN(request, "\r\n", 2)[0]
	headerParts := strings.Split(header, " ")

	return httpRequest{
		Method: headerParts[0],
		Url:    headerParts[1],
	}
}
