package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestInvalidMonitorFilter(t *testing.T) {
	handler := New(nil)
	for _, value := range []string{"0", "-1", "abc", "9999999999999999999999999"} {
		t.Run(value, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/results?monitor_id="+value, nil))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d", response.Code)
			}
		})
	}
}

func TestUnknownAPIEndpoint(t *testing.T) {
	response := httptest.NewRecorder()
	New(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestPagination(t *testing.T) {
	for _, path := range []string{"/api/monitors", "/api/results"} {
		for _, value := range []string{"0", "-1", "abc", "1000001"} {
			response := httptest.NewRecorder()
			New(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path+"?page="+value, nil))
			if response.Code != 400 {
				t.Fatalf("%s page=%s status=%d", path, value, response.Code)
			}
		}
	}
	for _, count := range []int{0, 49, 50, 51} {
		response := httptest.NewRecorder()
		paginated(response, make([]int, count), 2)
		var items []int
		if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		want := count
		if want > 50 {
			want = 50
		}
		if len(items) != want || response.Header().Get("X-Has-More") != strconv.FormatBool(count > 50) || response.Header().Get("X-Page") != "2" {
			t.Fatalf("count=%d headers=%v items=%d", count, response.Header(), len(items))
		}
	}
}

func TestInvalidMonitorBatch(t *testing.T) {
	for _, query := range []string{"monitor_ids=0", "monitor_ids=abc", "monitor_ids=1,-2", "monitor_ids=1&monitor_id=1"} {
		response := httptest.NewRecorder()
		New(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/results?"+query, nil))
		if response.Code != 400 {
			t.Fatalf("query=%s status=%d", query, response.Code)
		}
	}
}
