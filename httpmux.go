// Package httpmux provides an http request multiplexer.
package httpmux

import (
	"net/http"
	"path"
	"strings"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"

	"github.com/go-web/httpctx"
)

type (
	// Tree is the http request multiplexer backed by httprouter.Router.
	Tree struct {
		prefix string            // prefix for all paths
		mw     []Middleware      // list of mw set by Use
		routes map[string]*route // map of pattern to route for subtrees
		router *httprouter.Router
	}

	route struct {
		Method  string
		Handler httprouter.Handle
	}

	// Middleware is an http handler that can optionally
	// call the next handler in the chain based on
	// the request or any other conditions.
	Middleware func(next http.HandlerFunc) http.HandlerFunc

	// Config is the Tree configuration.
	Config struct {
		// Prefix is the prefix for all paths in the tree.
		// Empty value is allowed and defaults to "/".
		Prefix string

		// Middleware is the initial list of middlewares to be
		// automatically assigned to all handlers.
		Middleware []Middleware

		// Enables automatic redirection if the current route can't be matched but a
		// handler for the path with (without) the trailing slash exists.
		// For example if /foo/ is requested but a route only exists for /foo, the
		// client is redirected to /foo with http status code 301 for GET requests
		// and 307 for all other request methods.
		RedirectTrailingSlash bool

		// If enabled, the router tries to fix the current request path, if no
		// handle is registered for it.
		// First superfluous path elements like ../ or // are removed.
		// Afterwards the router does a case-insensitive lookup of the cleaned path.
		// If a handle can be found for this route, the router makes a redirection
		// to the corrected path with status code 301 for GET requests and 307 for
		// all other request methods.
		// For example /FOO and /..//Foo could be redirected to /foo.
		// RedirectTrailingSlash is independent of this option.
		RedirectFixedPath bool

		// If enabled, the router checks if another method is allowed for the
		// current route, if the current request can not be routed.
		// If this is the case, the request is answered with 'Method Not Allowed'
		// and HTTP status code 405.
		// If no other Method is allowed, the request is delegated to the NotFound
		// handler.
		HandleMethodNotAllowed bool

		// Configurable http.Handler which is called when no matching route is
		// found. If it is not set, http.NotFound is used.
		NotFound http.Handler

		// Configurable http.Handler which is called when a request
		// cannot be routed and HandleMethodNotAllowed is true.
		// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
		MethodNotAllowed http.Handler

		// Function to handle panics recovered from http handlers.
		// It should be used to generate a error page and return the http error code
		// 500 (Internal Server Error).
		// The handler can be used to keep your server from crashing because of
		// unrecovered panics.
		PanicHandler func(http.ResponseWriter, *http.Request, interface{})
	}
)

// New creates and initializes a new Tree using default settings.
func New() *Tree {
	return NewTree(&Config{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
	})
}

// NewTree creates and initializes a new Tree with the given config.
func NewTree(c *Config) *Tree {
	t := &Tree{
		prefix: c.Prefix,
		mw:     c.Middleware,
		routes: make(map[string]*route),
	}
	router := httprouter.New()
	router.RedirectTrailingSlash = c.RedirectTrailingSlash
	router.RedirectFixedPath = c.RedirectFixedPath
	router.HandleMethodNotAllowed = c.HandleMethodNotAllowed
	if c.NotFound != nil {
		router.NotFound = t.chain(c.NotFound.ServeHTTP)
	}
	if c.MethodNotAllowed != nil {
		router.MethodNotAllowed = t.chain(c.MethodNotAllowed.ServeHTTP)
	}
	router.PanicHandler = c.PanicHandler
	t.router = router
	return t
}

// DELETE is a shortcut for mux.Handle("DELETE", path, handle)
func (t *Tree) DELETE(pattern string, f http.HandlerFunc) { t.Handle("DELETE", pattern, f) }

// GET is a shortcut for mux.Handle("GET", path, handle)
func (t *Tree) GET(pattern string, f http.HandlerFunc) { t.Handle("GET", pattern, f) }

// HEAD is a shortcut for mux.Handle("HEAD", path, handle)
func (t *Tree) HEAD(pattern string, f http.HandlerFunc) { t.Handle("HEAD", pattern, f) }

// OPTIONS is a shortcut for mux.Handle("OPTIONS", path, handle)
func (t *Tree) OPTIONS(pattern string, f http.HandlerFunc) { t.Handle("OPTIONS", pattern, f) }

// PATCH is a shortcut for mux.Handle("PATCH", path, handle)
func (t *Tree) PATCH(pattern string, f http.HandlerFunc) { t.Handle("PATCH", pattern, f) }

// POST is a shortcut for mux.Handle("POST", path, handle)
func (t *Tree) POST(pattern string, f http.HandlerFunc) { t.Handle("POST", pattern, f) }

// PUT is a shortcut for mux.Handle("PUT", path, handle)
func (t *Tree) PUT(pattern string, f http.HandlerFunc) { t.Handle("PUT", pattern, f) }

// chain generates the middleware chain and appends f at the end.
func (t *Tree) chain(f http.HandlerFunc) http.HandlerFunc {
	var handler http.HandlerFunc
	for i := len(t.mw) - 1; i >= 0; i-- {
		handler = t.mw[i](f)
		f = handler
	}
	return f
}

// Handle registers a new request handler with the given method and pattern.
func (t *Tree) Handle(method, pattern string, f http.Handler) {
	f = t.chain(f.ServeHTTP)
	ff := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx := httpctx.Get(r)
		ctx = context.WithValue(ctx, paramsID, p)
		httpctx.Set(r, ctx)
		f.ServeHTTP(w, r)
		httpctx.Clear(r)
	}
	p := path.Join(t.prefix, pattern)
	if len(pattern) > 1 && pattern[len(pattern)-1] == '/' {
		p += "/"
	}
	t.routes[p] = &route{Method: method, Handler: ff}
	t.router.Handle(method, p, ff)
}

// HandleFunc registers a new request handler with the given method and pattern.
func (t *Tree) HandleFunc(method, pattern string, f http.HandlerFunc) {
	t.Handle(method, pattern, f)
}

// ServeFiles serves files from the given file system root.
//
// The pattern must end with "/*filepath" to have files served from
// the local path /path/to/dir/*filepath.
//
// For example, if root is "/etc" and *filepath is "passwd", the local
// file "/etc/passwd" is served. Because an http.FileServer is used
// internally it's possible that http.NotFound is called instead
// of httpmux's NotFound handler.
//
// To use the operating system's file system implementation, use
// http.Dir: mux.ServeFiles("/src/*filepath", http.Dir("/var/www")).
func (t *Tree) ServeFiles(pattern string, root http.FileSystem) {
	if !strings.HasSuffix(pattern, "/*filepath") {
		panic("pattern must end with /*filepath in path '" + pattern + "'")
	}
	fs := http.FileServer(root)
	t.GET(pattern, func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = Params(r).ByName("filepath")
		fs.ServeHTTP(w, r)
	})
}

// Use records the given middlewares to the internal chain.
func (t *Tree) Use(f ...Middleware) {
	t.mw = append(t.mw, f...)
}

// Append appends a subtree to the tree under the given pattern. All
// middleware from the root tree propagates to the subtree as well,
// however, subtree's url parameters are not available to the root
// tree's middleware. Also, subtree's configuration such as fallback
// handlers like NotFound and MethodNotAllowed are ignored by the
// root tree in favor of its own configuration.
func (t *Tree) Append(pattern string, subtree *Tree) {
	for pp, route := range subtree.routes {
		f := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			ff := func(w http.ResponseWriter, r *http.Request) {
				route.Handler(w, r, p)
				httpctx.Clear(r)
			}
			ff = t.chain(ff)
			ff(w, r)
		}
		pp = path.Join(t.prefix, pattern, pp)
		t.router.Handle(route.Method, pp, f)
	}
}

// ServeHTTP implements the http.Handler interface.
func (t *Tree) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.router.ServeHTTP(w, r)
}
