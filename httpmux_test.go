package httpmux

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestTree(t *testing.T) {
	mux := New()
	f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.DELETE("/", f)
	mux.GET("/", f)
	mux.HEAD("/", f)
	mux.OPTIONS("/", f)
	mux.PATCH("/:arg", f)
	mux.POST("/:arg", f)
	mux.PUT("/:arg", f)
	for i, method := range []string{
		"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT",
	} {
		r := &http.Request{
			Method: method,
			URL:    &url.URL{Path: "/"},
		}
		switch method {
		case "PATCH", "POST", "PUT":
			r.Body = ioutil.NopCloser(bytes.NewBuffer([]byte{1}))
			r.URL.Path += "arg"
		}
		w := &httptest.ResponseRecorder{}
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("Test %d: unexpected status code. Want %d, have %d",
				i, http.StatusOK, w.Code)
		}
	}
}

func TestSubtree(t *testing.T) {
	root := New()
	root.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Hello", "world")
			next(w, r)
		}
	})
	tree := New()
	tree.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("X-Hello") == "world" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})
	root.Append("/test", tree)
	r := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test/foobar"},
	}
	w := &httptest.ResponseRecorder{}
	root.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status code. Want %d, have %d",
			http.StatusOK, w.Code)
	}

}

func testmw(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Hello", "world")
		next.ServeHTTP(w, r)
	}
}

func TestMiddleware(t *testing.T) {
	mux := New()
	mux.Use(testmw)
	respc := make(chan bool, 1)
	f := func(w http.ResponseWriter, r *http.Request) {
		respc <- w.Header().Get("X-Hello") == "world"
	}
	mux.GET("/", http.HandlerFunc(f))
	r := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
	}
	w := &httptest.ResponseRecorder{}
	mux.ServeHTTP(w, r)
	select {
	case ok := <-respc:
		if !ok {
			t.Fatalf("Middleware not executed.")
		}
	default:
		t.Fatalf("Handler not executed.")
	}
}

func TestServeFiles(t *testing.T) {
	i := 0
	p := map[string]struct{ Dir, URL string }{
		"/*filepath":        {".", "/httpmux.go"},
		"/foobar/*filepath": {".", "/foobar/httpmux.go"},
	}
	for pattern, cfg := range p {
		mux := New()
		mux.ServeFiles(pattern, http.Dir(cfg.Dir))
		w := &httptest.ResponseRecorder{}
		r := &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: cfg.URL},
		}
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("Test %d: Unexpected status. Want %d, have %d",
				i, http.StatusOK, w.Code)
		}
		i++
	}
}
