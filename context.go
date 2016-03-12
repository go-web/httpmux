package httpmux

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"

	"github.com/go-web/httpctx"
)

// Context is a middleware for httpmux that associates context with
// the request using httpctx.
func Context(ctx context.Context) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		f := func(w http.ResponseWriter, r *http.Request) {
			httpctx.Set(r, ctx)
			next(w, r)
		}
		return http.HandlerFunc(f)
	}
}

type paramsType int

var paramsID paramsType

// Params returns the httprouter.Params from the request context.
func Params(r *http.Request) httprouter.Params {
	if p, ok := httpctx.Get(r).Value(paramsID).(httprouter.Params); ok {
		return p
	}
	return httprouter.Params{}
}
