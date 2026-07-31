package static

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestHandler(t *testing.T) {
	sub := fstest.MapFS{
		"index.html": {Data: []byte("<html>home</html>")},
		"200.html":   {Data: []byte("<html>spa</html>")},
		"app.js":     {Data: []byte("console.log('ok')")},
	}

	handler := newHandler(sub)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "root serves index without redirect",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "<html>home</html>",
		},
		{
			name:       "index html path serves index without redirect",
			path:       "/index.html",
			wantStatus: http.StatusOK,
			wantBody:   "<html>home</html>",
		},
		{
			name:       "nested route falls back to spa shell",
			path:       "/licenses",
			wantStatus: http.StatusOK,
			wantBody:   "<html>spa</html>",
		},
		{
			name:       "existing asset is served",
			path:       "/app.js",
			wantStatus: http.StatusOK,
			wantBody:   "console.log('ok')",
		},
		{
			name:       "api paths are not handled",
			path:       "/api/v1/licenses",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" {
				body, err := io.ReadAll(rec.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if string(body) != tt.wantBody {
					t.Fatalf("body = %q, want %q", body, tt.wantBody)
				}
			}
		})
	}
}
