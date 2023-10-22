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

type Request struct {
	Method   string
	Path     string
	Protocol string
	Body     string
	Headers  Headers
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
	request, err := parseRequest(string(requestBytes))
	if err != nil {
		fmt.Println("Error parsing request: ", err.Error())
		os.Exit(1)
	}

	response, err := getResponse(request)

	// write the response to the connection
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("failed to write response: ", err.Error())
		os.Exit(1)
	}
}

func getResponse(request *Request) (string, error) {
	var err error
	var response string
	if request == nil {
		return "", fmt.Errorf("empty request")
	}
	if request.Path == "/" {
		response = okMessage
	} else if strings.HasPrefix(request.Path, "/echo/") {
		// Echo the message
		response, err = getEchoResponse(request.Path)
		if err != nil {
			fmt.Println("Fail to get response: ", err.Error())
			os.Exit(1)
		}
	} else if strings.HasPrefix(request.Path, "/user-agent") && request.Method == http.MethodGet {
		if userAgent, ok := request.Headers["User-Agent"]; ok {
			response = getUserAgentResponse(userAgent)
		}
	} else if strings.HasPrefix(request.Path, "/files") {
		if request.Method == http.MethodGet {
			response, err = getFileResponse(request.Path)
		} else if request.Method == http.MethodPost {
			response, err = postFileResponse(request)
		}
		if err != nil {
			fmt.Println("Fail to get response: ", err.Error())
			os.Exit(1)
		}
	} else {
		response = notFoundMessage
	}
	return response, err
}

func postFileResponse(request *Request) (string, error) {
	// get the filename
	filename := strings.TrimPrefix(request.Path, "/files/")
	filePath := filepath.Join(*directory, filename)

	// Write the contents of the request body at the specified filepath.
	err := os.WriteFile(filePath, []byte(request.Body), 0644)
	if err != nil {
		return "", err
	}

	// return a 201 OK response
	return "HTTP/1.1 201 Created\r\n\r\n", nil
}

func getFileResponse(path string) (string, error) {
	// get the filename
	filename := strings.TrimPrefix(path, "/files/")
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

func parseRequest(request string) (*Request, error) {
	if request == "" {
		return nil, fmt.Errorf("empty string")
	}
	lines := strings.Split(request, "\r\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("invalid request")
	}
	r, err := getRequest(lines)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func getRequest(lines []string) (*Request, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty request")
	}
	// initialize the headers
	headers := make(Headers)

	// get the start line
	words := strings.Split(lines[0], " ")
	if len(words) != 3 {
		return nil, fmt.Errorf("invalid string")
	}
	request := &Request{
		Method:   words[0],
		Path:     words[1],
		Protocol: words[2],
	}

	// get the headers and the body
	for i, line := range lines[1:] {
		if line == "" {
			// Headers and body are separated by an empty line
			request.Body = strings.Join(lines[i+1:], "\r\n")
			break
		}
		before, after, ok := strings.Cut(line, ":")
		if ok {
			headers[strings.TrimSpace(before)] = strings.TrimSpace(after)
		}
	}
	return request, nil

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

func getEchoResponse(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	// parse the path
	message := getMessage(path)
	if message == "" {
		return "", fmt.Errorf("empty message")
	}
	return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message), nil
}

func getMessage(path string) string {
	splits := strings.Split(path, "echo/")
	return splits[len(splits)-1]
}
