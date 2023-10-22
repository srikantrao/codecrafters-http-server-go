package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	requestStr := "GET /index.html HTTP/1.1\r\n\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1"
	request, err := parseRequest(requestStr)
	assert.Nil(t, err)

	// set up the expected result for the start line
	expectedMethod := http.MethodGet
	expectedPath := "/index.html"
	expectedProtocol := "HTTP/1.1"
	assert.Equal(t, request.Method, expectedMethod, "http method does not match")
	assert.Equal(t, request.Path, expectedPath, "expected %v actual %v", expectedPath, request.Path)
	assert.Equal(t, request.Protocol, expectedProtocol, "expected %v actual %v", expectedProtocol, request.Protocol)

	// set up the expected result for the headers
	expectedHost := "localhost:4221"
	expectedUserAgent := "curl/7.64.1"

	assert.Equal(t, request.Headers["Host"], expectedHost, "expected %v actual %v", expectedHost, request.Headers["Host"])
	assert.Equal(t, request.Headers["User-Agent"], expectedUserAgent, "expected %v actual %v", expectedUserAgent, request.Headers["User-Agent"])
}

func TestEchoRequest(t *testing.T) {
	requestStr := "GET /echo/abc HTTP/1.1\r\n\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1"
	request, err := parseRequest(requestStr)
	assert.Nil(t, err)
	responseStr, err := getEchoResponse(request.Path)
	assert.Nil(t, err)

	//set up the expected result for the response string
	assert.Equal(t, responseStr, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 3\r\n\r\nabc",
		"expected %v actual %v", "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 3\r\n\r\nabc", responseStr)
}

func TestUserAgentRequest(t *testing.T) {
	requestStr := "GET /user-agent HTTP/1.1\r\n\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1"
	request, err := parseRequest(requestStr)
	assert.Nil(t, err)
	responseStr := getUserAgentResponse(request.Headers["User-Agent"])
	assert.Nil(t, err)

	expectedResponse := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 11\r\n\r\ncurl/7.64.1"
	//set up the expected result for the response string
	assert.Equal(t, responseStr, expectedResponse,
		"expected %v actual %v", expectedResponse, responseStr)
}

func TestUserAgentRequest2(t *testing.T) {
	requestStr := "GET /user-agent HTTP/1.1\r\n\r\nHost: localhost:4221\r\nUser-Agent: humpty/vanilla-dumpty\r\nAccept-Encoding: gzip"
	request, err := parseRequest(requestStr)
	assert.Nil(t, err)
	response, err := getResponse(request)
	assert.Nil(t, err)

	expectedResponse := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 21\r\n\r\nhumpty/vanilla-dumpty"
	//set up the expected result for the response string
	assert.Equal(t, response, expectedResponse,
		"expected %v actual %v", expectedResponse, response)
}
