package httpmux

import (
	"net/http"
	"reflect"
	"testing"
)

func TestConfigOptions(t *testing.T) {
	have := DefaultConfig
	want := &Config{
		Prefix:                 "/foobar",
		Middleware:             nil,
		RedirectTrailingSlash:  false,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		NotFound:               http.NewServeMux(),
		MethodNotAllowed:       http.NewServeMux(),
		PanicHandler:           nil,
	}
	for _, f := range []ConfigOption{
		WithPrefix(want.Prefix),
		WithMiddleware(want.Middleware...),
		WithRedirectTrailingSlash(want.RedirectTrailingSlash),
		WithRedirectFixedPath(want.RedirectFixedPath),
		WithHandleMethodNotAllowed(want.HandleMethodNotAllowed),
		WithNotFound(want.NotFound),
		WithMethodNotAllowed(want.MethodNotAllowed),
		WithPanicHandler(want.PanicHandler),
	} {
		f.Set(&have)
	}
	if !reflect.DeepEqual(have, *want) {
		t.Fatalf("data mismatch.\nwant: %#v\nhave: %#v\n", *want, have)
	}
}
