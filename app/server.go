package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type httpRequest struct {
	Method         string
	Url            string
	PathParam      string
	AcceptEncoding string
	UserAgent      string
	Body           string
}

func main() {
	log.Printf("Listening on 4221...")
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {

		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go requestHandler(conn)
	}
}

func requestHandler(conn net.Conn) {
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
	var httpRequest httpRequest
	request := string(buffer)
	log.Printf("Request: %v", request)
	requestFields := strings.Fields(request)
	httpRequest.Method = requestFields[0]
	urlParts := strings.Split(requestFields[1], "/")

	var count int
	var body string
	for i, s := range requestFields {
		if strings.ToLower(s) == "content-type:" {
			count = 2 + i
		}
	}

	for i := count; i < len(requestFields); i++ {
		body += requestFields[i]
		if i < len(requestFields)-1 {
			body += " "
		}
	}
	log.Println("body - > " + body)
	httpRequest.Body = body

	switch len(urlParts) {
	case 2:
		httpRequest.Url = urlParts[1]
	case 3:
		httpRequest.Url = urlParts[1]
		httpRequest.PathParam = urlParts[2]
	}

	for i, r := range requestFields {
		switch strings.ToLower(r) {
		case "user-agent:":
			httpRequest.UserAgent = requestFields[i+1]
		case "accept-encoding:":
			httpRequest.AcceptEncoding = requestFields[i+1]
			for j := len(requestFields) - i; j < len(requestFields); j++ {
				if strings.ToLower(strings.Trim(requestFields[j], ",")) == "gzip" {
					httpRequest.AcceptEncoding = "gzip"
				}
			}
		}

	}

	responseHander(httpRequest, conn)
}

func responseHander(req httpRequest, conn net.Conn) {
	defer conn.Close()

	var response string
	switch req.Url {
	case "":
		response = "HTTP/1.1 200 OK\r\n\r\n"
	case "/":
		response = "HTTP/1.1 200 OK\r\n\r\n"
	case "user-agent":
		length := len(req.UserAgent)
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %v\r\n\r\n%v", length, req.UserAgent)
	case "echo":
		// res := gZip(req.PathParam)
		str := req.PathParam
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		if _, err := w.Write([]byte(str)); err != nil {
			log.Fatal(err)
		}
		if err := w.Close(); err != nil {
			log.Fatal(err)
		}

		if req.AcceptEncoding == "gzip" {
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\nContent-Encoding: gzip\r\n\r\n%s", len(b.Bytes()), b.Bytes())
			// a := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: %v\r\nContent-Length: %v\r\n\r\n%v", req.AcceptEncoding, len(req.PathParam), res)
			// response = a
			log.Println(response)
		} else {
			// response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %v\r\n\r\n%v", len(req.PathParam), req.PathParam)

			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(req.PathParam), str)
		}

	case "files":
		if req.Method == "GET" {
			length, data, err := readFile(req.PathParam)
			if err != nil {
				response = ("HTTP/1.1 404 Not Found\r\n\r\n")
			} else {
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %v\r\n\r\n%v", length, data)
			}
		}

		if req.Method == "POST" {
			data := []byte(req.Body)
			dir := "/tmp/data/codecrafters.io/http-server-tester/" + req.PathParam
			os.Create(dir)
			os.WriteFile(dir, data, 777)
			response = fmt.Sprintf("HTTP/1.1 201 Created\r\n\r\n")
		}

	default:
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	_, err := conn.Write([]byte(response))

	if err != nil {
		fmt.Println("Error writing to connection")
		os.Exit(1)
	}
}

func gZip(req string) string {
	var buffer bytes.Buffer
	w := gzip.NewWriter(&buffer)
	w.Write([]byte(req))
	w.Close()
	req = buffer.String()
	return req
}

func returnInHexFormat(data string) string {
	var formattedString string
	var count int
	for i, s := range data {
		count++
		if i < len(data)-1 {
			formattedString += string(s)
			if count%2 == 0 {
				formattedString += " "
			}
		} else {
			formattedString += string(s)
		}
	}
	return formattedString
}
func findFilesInTmp() map[string]string {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal("err while getting current path")
	}

	files := make(map[string]string)
	path = path + "/tmp/"
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if filepath.Base(path) != "tmp" {
			files[path] = filepath.Base(path)
			if err != nil {
				log.Println("ERROR:", err)
			}
		}
		return nil
	})
	return files
}
func readFile(fileName string) (length int, data string, e error) {
	log.Println("file name :" + fileName)
	file, err := os.ReadFile("/tmp/data/codecrafters.io/http-server-tester/" + fileName)
	if err != nil {
		log.Println("err while reading file")
	}
	return len(file), string(file), err
}
