package registration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer setup-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Result{MachineID: "machine-id", Token: "machine-token"})
	}))
	defer server.Close()
	result, err := (Client{ServerURL: server.URL, SetupKey: "setup-key"}).Register(context.Background(), "machine-id", "workstation")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != "machine-token" {
		t.Fatalf("result = %#v", result)
	}
}
