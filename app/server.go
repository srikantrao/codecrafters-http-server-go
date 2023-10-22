package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

var (
	directory = flag.String("directory", ".", "the directory to look for the files specified")
)

func main() {

	flag.Parse()
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	defer l.Close()
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	for {
		// accept a new connection...
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleRequest(conn)

	}

}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	// Read the request string
	requestBytes, err := readRequest(conn)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
	}
	router(conn, requestBytes)
}

func router(conn net.Conn, requestBytes []byte) {
	// Parse the request
	startLine, headers, err := parseRequest(string(requestBytes))

	response, err := getResponse(startLine, headers)

	// write the response to the connection
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("failed to write response: ", err.Error())
		os.Exit(1)
	}
}

func getResponse(startLine *StartLine, headers Headers) (string, error) {
	var err error
	var response string
	if startLine.Path == "/" {
		response = okMessage
	} else if strings.HasPrefix(startLine.Path, "/echo/") {
		// Echo the message
		response, err = getEchoResponse(startLine)
		if err != nil {
			fmt.Println("Fail to get response: ", err.Error())
			os.Exit(1)
		}
	} else if strings.HasPrefix(startLine.Path, "/user-agent") && startLine.Method == http.MethodGet {
		if userAgent, ok := headers["User-Agent"]; ok {
			response = getUserAgentResponse(userAgent)
		}
	} else if strings.HasPrefix(startLine.Path, "/files") && startLine.Method == http.MethodGet {
		response, err = getFileResponse(startLine)
		if err != nil {
			fmt.Println("Fail to get response: ", err.Error())
			os.Exit(1)
		}
	} else {
		response = notFoundMessage
	}
	return response, err
}

func getFileResponse(line *StartLine) (string, error) {
	// get the filename
	filename := strings.TrimPrefix(line.Path, "/files/")
	filePath := filepath.Join(*directory, filename)

	// Check if the file exists
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// The file does not exist.
			return notFoundMessage, nil
		} else {
			// An error occurred while checking if the file exists.
			return "", err
		}
	}

	// Open the file
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read the contents of the file.
	contents, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(contents), contents), nil

}

func getUserAgentResponse(agent string) string {
	return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(agent), agent)
}

func parseRequest(request string) (*StartLine, Headers, error) {
	if request == "" {
		return nil, nil, fmt.Errorf("empty string")
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

func readRequest(conn net.Conn) ([]byte, error) {
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

func getEchoResponse(startLine *StartLine) (string, error) {
	if startLine.Path == "" {
		return "", fmt.Errorf("empty path")
	}
	// parse the path
	message := getMessage(startLine)
	if message == "" {
		return "", fmt.Errorf("empty message")
	}
	return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message), nil
}

func getMessage(startLine *StartLine) string {
	splits := strings.Split(startLine.Path, "echo/")
	return splits[len(splits)-1]
}
