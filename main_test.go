package main

import (
	"testing"
	"fmt"
	"net/http/httptest"
	"io"
	"net/http"
)


var (
	server   *httptest.Server
	reader   io.Reader
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
	go checkPoller(checkSlice)


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



