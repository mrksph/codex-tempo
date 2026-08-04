package wakapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".wakatime.cfg")
	if err := os.WriteFile(path, []byte("[settings]\napi_url = http://wakapi.test/api\napi_key = secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credentials, err := LoadCredentials(path)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.APIURL != "http://wakapi.test/api" || credentials.APIKey != "secret" {
		t.Fatalf("credentials = %#v", credentials)
	}
}

func TestExportBuildsContiguousIntervalsWithoutContent(t *testing.T) {
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("secret"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expectedAuth {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/compat/wakatime/v1/users/current/all_time_since_today":
			fmt.Fprint(w, `{"data":{"range":{"start_date":"2026-04-29","end_date":"2026-04-30"}}}`)
		case "/api/compat/wakatime/v1/users/current/heartbeats":
			if r.URL.Query().Get("date") == "2026-04-29" {
				fmt.Fprint(w, `{"data":[{"id":"1","time":1777413600,"project":"alpha","entity":"SECRET"},{"id":"2","time":1777413660,"project":"alpha"},{"id":"3","time":1777413720,"project":"beta"},{"id":"4","time":1777414400,"project":"beta"}]}`)
			} else {
				fmt.Fprint(w, `{"data":[{"id":"5","time":1777500000,"project":"beta"},{"id":"6","time":1777500030,"project":"beta"}]}`)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	export, err := (Client{Credentials: Credentials{APIURL: server.URL + "/api", APIKey: "secret"}}).Export(context.Background(), "", "", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if export.Heartbeats != 6 || len(export.Intervals) != 2 || export.Duration != 150*time.Second {
		t.Fatalf("export = %#v", export)
	}
	if export.Intervals[0].Project != "alpha" || export.Intervals[0].EndedAt.Sub(export.Intervals[0].StartedAt) != 120*time.Second {
		t.Fatalf("first interval = %#v", export.Intervals[0])
	}
}
