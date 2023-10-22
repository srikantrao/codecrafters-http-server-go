package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	// Uncomment this block to pass the first stage
	// "net"
	// "os"
)

type StartLine struct {
	Method   string
	Path     string
	Protocol string
}

type Headers map[string]string

const (
	okMessage       = "HTTP/1.1 200 OK\r\n\r\n"
	notFoundMessage = "HTTP/1.1 404 Not Found\r\n\r\n"
)

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	defer l.Close()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	// Read the request string
	requestBytes, err := ReadRequest(conn)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
	}
	// Parse the request
	startLine, _, err := parseRequest(string(requestBytes))

	if startLine.Path == "/" {
		_, err = conn.Write([]byte(okMessage))
	} else {
		_, err = conn.Write([]byte(notFoundMessage))
	}
	if err != nil {
		fmt.Println("Fail to write response: ", err.Error())
		os.Exit(1)
	}
}

func parseRequest(request string) (*StartLine, Headers, error) {
	if request == "" {
		return nil, nil, fmt.Errorf("Empty string")
	}
	lines := strings.Split(request, "\r\n")
	if len(lines) == 0 {
		return nil, nil, fmt.Errorf("invalid request")
	}
	startLine, err := getStartLine(lines[0])
	if err != nil {
		return nil, nil, err
	}
	headers := make(Headers)
	if len(lines) > 1 {
		headers = getHeaders(lines[1:])
	}
	return startLine, headers, nil
}

func getStartLine(line string) (*StartLine, error) {
	if line == "" {
		return nil, fmt.Errorf("empty string")
	}
	words := strings.Split(line, " ")
	if len(words) == 3 {
		return &StartLine{words[0], words[1], words[2]}, nil
	}
	return nil, fmt.Errorf("invalid string")
}

func getHeaders(lines []string) Headers {
	headers := make(Headers)
	for _, line := range lines {
		before, after, ok := strings.Cut(line, ":")
		if ok {
			headers[strings.TrimSpace(before)] = strings.TrimSpace(after)
		}
	}
	return headers
}

func ReadRequest(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 102400)
	n, err := conn.Read(buf)
	// error that is not EOF
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n < len(buf) || errors.Is(err, io.EOF) {
		return buf[:n], nil
	}
	return buf, nil
}
