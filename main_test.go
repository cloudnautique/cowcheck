package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"os"
	"github.com/Sirupsen/logrus"
)

var (
	server *httptest.Server
	reader io.Reader
)

type FakeCheck struct {
	Check
}

func NewFakeCheck() *FakeCheck {
	return &FakeCheck{
		Check{
			name:          "FakeCheck",
			description:   "FakeCheck",
			currentStatus: true,
		},
	}
}

func (c *FakeCheck) eval() bool {
	fmt.Println("Evaluating check", c.name)
	return true
}

func TestCheckPoller(t *testing.T) {
	checkSlice = append(checkSlice, NewFakeCheck())
	go checkPoller(checkSlice, 2)

}

func TestHTTP(t *testing.T) {
	// Borrowing example from https://elithrar.github.io/article/testing-http-handlers-go/

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(checkState)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `Everything OK`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
func TestParseConfig(t *testing.T) {
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("POLL_INTERVAL", "25")
	cfg := parseConfig()
	if cfg.logLevel == logrus.DebugLevel {
		t.Logf("Found expected value for logLevel")
	} else {
		t.Errorf("Error: Did not find expected value for logLevel. Expected logrus.DebugLevel")
	}
	if cfg.pollInterval == 25 {
		t.Logf("Found expected value for pollInterval")

	} else {
		t.Errorf("Did not find expected value for pollInverval")

	}

}


