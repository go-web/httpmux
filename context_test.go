package httpmux

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/net/context"

	"github.com/go-web/httpctx"
)

func TestContext(t *testing.T) {
	w := &httptest.ResponseRecorder{}
	r := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
		Body:   ioutil.NopCloser(&bytes.Buffer{}),
	}
	x := context.Background()
	mw := Context(x)
	mw(http.NewServeMux().ServeHTTP)(w, r)
	y := httpctx.Get(r)
	if x != y {
		t.Fatalf("Unexpected ctx. Want %v, have %v", x, y)
	}
}
