package watchdog

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCheckInternet tests the checkInternet function with both a successful
// and a failing mock server to verify its behavior.
func TestCheckInternet(t *testing.T) {
	// --- Test Case 1: Server is online and returns 200 OK ---

	// Create a mock HTTP server that always responds with 200 OK.
	serverOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// defer is used to ensure the server is closed after the test finishes.
	defer serverOK.Close()

	// Call the function we are testing, pointing it to our mock server's URL.
	isOnline := checkInternet(serverOK.URL)

	// Assert that the result is what we expect.
	if !isOnline {
		t.Errorf("checkInternet() returned false for an OK server, expected true")
	}

	// --- Test Case 2: Server is online but returns a 500 Internal Server Error ---

	// Create another mock server that returns an error status code.
	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverError.Close()

	// Call the function again with the error-producing server.
	isOnline = checkInternet(serverError.URL)

	// Assert that the function correctly reports the connection as "down".
	if isOnline {
		t.Errorf("checkInternet() returned true for an erroring server, expected false")
	}

	// --- Test Case 3: Server is offline (unreachable) ---

	// Call the function with a non-existent URL.
	isOnline = checkInternet("http://localhost:12345")

	// Assert that the function returns false when the server is unreachable.
	if isOnline {
		t.Errorf("checkInternet() returned true for an unreachable server, expected false")
	}
}
