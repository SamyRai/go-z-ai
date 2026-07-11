package coding

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateKeyValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer srv.Close()

	if err := validateKeyAt(srv.URL, "good", &http.Client{Timeout: 5 * time.Second}); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateKeyInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := validateKeyAt(srv.URL, "bad", &http.Client{Timeout: 5 * time.Second})
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestValidateKeyUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := validateKeyAt(srv.URL, "k", &http.Client{Timeout: 5 * time.Second})
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("500 should not be classified as invalid key: %v", err)
	}
}
