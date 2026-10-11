package datafs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimeseriesAndServerEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth %s", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/sites/site-1/timeseries":
			if r.URL.Query().Get("from") != "2026-09-12" || r.URL.Query().Get("interval") != "day" {
				t.Errorf("query %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"from":"2026-09-12","to":"2026-10-11","interval":"day","totals":{"visitors":1,"pageviews":2,"visits":1,"bounce_rate":0,"visit_duration_seconds":0},"points":[]}`)
		case "/api/v1/server-event":
			body, _ := io.ReadAll(r.Body)
			if string(body) == "" || r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("body %s type %s", body, r.Header.Get("Content-Type"))
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"event_id":"evt-1"}`)
		default:
			t.Errorf("path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, APIKey: "secret"}
	series, err := c.Timeseries(context.Background(), "site-1", "2026-09-12", "2026-10-11", "day")
	if err != nil {
		t.Fatal(err)
	}
	if series.Totals.Visitors != 1 || series.Totals.Pageviews != 2 {
		t.Fatalf("totals %+v", series.Totals)
	}
	id, err := c.ServerEvent(context.Background(), ServerEvent{Domain: "example.com", Name: "signup"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "evt-1" {
		t.Fatalf("id %s", id)
	}
}

func TestErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad_range","message":"from > to"}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "secret"}
	_, err := c.Realtime(context.Background(), "site-1")
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Code != "bad_range" || apiErr.Status != 400 {
		t.Fatalf("err %#v", err)
	}
}
