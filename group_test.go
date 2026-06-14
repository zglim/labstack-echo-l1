// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2015 LabStack LLC and Echo contributors

package echo

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup_withoutRouteWillNotExecuteMiddleware(t *testing.T) {
	e := New()

	called := false
	mw := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = true
			return c.NoContent(http.StatusTeapot)
		}
	}
	// even though group has middleware it will not be executed when there are no routes under that group
	_ = e.Group("/group", mw)

	status, body := request(http.MethodGet, "/group/nope", e)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, `{"message":"Not Found"}`+"\n", body)

	assert.False(t, called)
}

func TestGroup_withRoutesWillNotExecuteMiddlewareFor404(t *testing.T) {
	e := New()

	called := false
	mw := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = true
			return c.NoContent(http.StatusTeapot)
		}
	}
	// even though group has middleware and routes when we have no match on some route the middlewares for that
	// group will not be executed
	g := e.Group("/group", mw)
	g.GET("/yes", handlerFunc)

	status, body := request(http.MethodGet, "/group/nope", e)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, `{"message":"Not Found"}`+"\n", body)

	assert.False(t, called)
}

func TestGroup_multiLevelGroup(t *testing.T) {
	e := New()

	api := e.Group("/api")
	users := api.Group("/users")
	users.GET("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	status, body := request(http.MethodGet, "/api/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroupFile(t *testing.T) {
	e := New()
	g := e.Group("/group")
	g.File("/walle", "_fixture/images/walle.png")
	expectedData, err := os.ReadFile("_fixture/images/walle.png")
	assert.Nil(t, err)
	req := httptest.NewRequest(http.MethodGet, "/group/walle", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, expectedData, rec.Body.Bytes())
}

func TestGroupRouteMiddleware(t *testing.T) {
	// Ensure middleware slices are not re-used
	e := New()
	g := e.Group("/group")
	h := func(*Context) error { return nil }
	m1 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return next(c)
		}
	}
	m2 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return next(c)
		}
	}
	m3 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return next(c)
		}
	}
	m4 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return c.NoContent(404)
		}
	}
	m5 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return c.NoContent(405)
		}
	}
	g.Use(m1, m2, m3)
	g.GET("/404", h, m4)
	g.GET("/405", h, m5)

	c, _ := request(http.MethodGet, "/group/404", e)
	assert.Equal(t, 404, c)
	c, _ = request(http.MethodGet, "/group/405", e)
	assert.Equal(t, 405, c)
}

func TestGroupRouteMiddlewareWithMatchAny(t *testing.T) {
	// Ensure middleware and match any routes do not conflict
	e := New()
	g := e.Group("/group")
	m1 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return next(c)
		}
	}
	m2 := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			return c.String(http.StatusOK, c.RouteInfo().Path)
		}
	}
	h := func(c *Context) error {
		return c.String(http.StatusOK, c.RouteInfo().Path)
	}
	g.Use(m1)
	g.GET("/help", h, m2)
	g.GET("/*", h, m2)
	g.GET("", h, m2)
	e.GET("unrelated", h, m2)
	e.GET("*", h, m2)

	_, m := request(http.MethodGet, "/group/help", e)
	assert.Equal(t, "/group/help", m)
	_, m = request(http.MethodGet, "/group/help/other", e)
	assert.Equal(t, "/group/*", m)
	_, m = request(http.MethodGet, "/group/404", e)
	assert.Equal(t, "/group/*", m)
	_, m = request(http.MethodGet, "/group", e)
	assert.Equal(t, "/group", m)
	_, m = request(http.MethodGet, "/other", e)
	assert.Equal(t, "/*", m)
	_, m = request(http.MethodGet, "/", e)
	assert.Equal(t, "/*", m)

}

func TestGroup_CONNECT(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.CONNECT("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodConnect, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodConnect+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodConnect, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_DELETE(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.DELETE("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodDelete, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodDelete+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodDelete, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_HEAD(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.HEAD("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodHead, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodHead+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodHead, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_OPTIONS(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.OPTIONS("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodOptions, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodOptions+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodOptions, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_PATCH(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.PATCH("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodPatch, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodPatch+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodPatch, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_POST(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.POST("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodPost, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodPost+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodPost, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_PUT(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.PUT("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodPut, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodPut+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodPut, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_TRACE(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.TRACE("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodTrace, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, http.MethodTrace+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodTrace, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_RouteNotFound(t *testing.T) {
	var testCases = []struct {
		expectRoute any
		name        string
		whenURL     string
		expectCode  int
	}{
		{
			name:        "404, route to static not found handler /group/a/c/xx",
			whenURL:     "/group/a/c/xx",
			expectRoute: "GET /group/a/c/xx",
			expectCode:  http.StatusNotFound,
		},
		{
			name:        "404, route to path param not found handler /group/a/:file",
			whenURL:     "/group/a/echo.exe",
			expectRoute: "GET /group/a/:file",
			expectCode:  http.StatusNotFound,
		},
		{
			name:        "404, route to any not found handler /group/*",
			whenURL:     "/group/b/echo.exe",
			expectRoute: "GET /group/*",
			expectCode:  http.StatusNotFound,
		},
		{
			name:        "200, route /group/a/c/df to /group/a/c/df",
			whenURL:     "/group/a/c/df",
			expectRoute: "GET /group/a/c/df",
			expectCode:  http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := New()
			g := e.Group("/group")

			okHandler := func(c *Context) error {
				return c.String(http.StatusOK, c.Request().Method+" "+c.Path())
			}
			notFoundHandler := func(c *Context) error {
				return c.String(http.StatusNotFound, c.Request().Method+" "+c.Path())
			}

			g.GET("/", okHandler)
			g.GET("/a/c/df", okHandler)
			g.GET("/a/b*", okHandler)
			g.PUT("/*", okHandler)

			g.RouteNotFound("/a/c/xx", notFoundHandler)  // static
			g.RouteNotFound("/a/:file", notFoundHandler) // param
			g.RouteNotFound("/*", notFoundHandler)       // any

			req := httptest.NewRequest(http.MethodGet, tc.whenURL, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectCode, rec.Code)
			assert.Equal(t, tc.expectRoute, rec.Body.String())
		})
	}
}

func TestGroup_Any(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.Any("/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK from ANY")
	})

	assert.Equal(t, RouteAny, ri.Method)
	assert.Equal(t, "/users/activate", ri.Path)
	assert.Equal(t, RouteAny+":/users/activate", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodTrace, "/users/activate", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK from ANY`, body)
}

func TestGroup_Match(t *testing.T) {
	e := New()

	myMethods := []string{http.MethodGet, http.MethodPost}
	users := e.Group("/users")
	ris := users.Match(myMethods, "/activate", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})
	assert.Len(t, ris, 2)

	for _, m := range myMethods {
		status, body := request(m, "/users/activate", e)
		assert.Equal(t, http.StatusTeapot, status)
		assert.Equal(t, `OK`, body)
	}
}

func TestGroup_MatchWithErrors(t *testing.T) {
	e := New()

	users := e.Group("/users")
	users.GET("/activate", func(c *Context) error {
		return c.String(http.StatusOK, "OK")
	})
	myMethods := []string{http.MethodGet, http.MethodPost}

	errs := func() (errs []error) {
		defer func() {
			if r := recover(); r != nil {
				if tmpErr, ok := r.([]error); ok {
					errs = tmpErr
					return
				}
				panic(r)
			}
		}()

		users.Match(myMethods, "/activate", func(c *Context) error {
			return c.String(http.StatusTeapot, "OK")
		})
		return nil
	}()
	assert.Len(t, errs, 1)
	assert.EqualError(t, errs[0], "GET /users/activate: adding duplicate route (same method+path) is not allowed")

	for _, m := range myMethods {
		status, body := request(m, "/users/activate", e)

		expect := http.StatusTeapot
		if m == http.MethodGet {
			expect = http.StatusOK
		}
		assert.Equal(t, expect, status)
		assert.Equal(t, `OK`, body)
	}
}

func TestGroup_Static(t *testing.T) {
	e := New()

	g := e.Group("/books")
	ri := g.Static("/download", "_fixture")
	assert.Equal(t, http.MethodGet, ri.Method)
	assert.Equal(t, "/books/download*", ri.Path)
	assert.Equal(t, "GET:/books/download*", ri.Name)
	assert.Equal(t, []string{"*"}, ri.Parameters)

	req := httptest.NewRequest(http.MethodGet, "/books/download/index.html", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.True(t, strings.HasPrefix(body, "<!doctype html>"))
}

func TestGroup_StaticMultiTest(t *testing.T) {
	var testCases = []struct {
		name                  string
		givenPrefix           string
		givenRoot             string
		whenURL               string
		expectHeaderLocation  string
		expectBodyStartsWith  string
		expectBodyNotContains string
		expectStatus          int
	}{
		{
			name:                 "ok",
			givenPrefix:          "/images",
			givenRoot:            "_fixture/images",
			whenURL:              "/test/images/walle.png",
			expectStatus:         http.StatusOK,
			expectBodyStartsWith: string([]byte{0x89, 0x50, 0x4e, 0x47}),
		},
		{
			name:                 "ok, without prefix",
			givenPrefix:          "",
			givenRoot:            "_fixture/images",
			whenURL:              "/testwalle.png", // `/test` + `*` creates route `/test*` witch matches `/testwalle.png`
			expectStatus:         http.StatusOK,
			expectBodyStartsWith: string([]byte{0x89, 0x50, 0x4e, 0x47}),
		},
		{
			name:                 "nok, without prefix does not serve dir index",
			givenPrefix:          "",
			givenRoot:            "_fixture/images",
			whenURL:              "/test/", // `/test` + `*` creates route `/test*`
			expectStatus:         http.StatusNotFound,
			expectBodyStartsWith: "{\"message\":\"Not Found\"}\n",
		},
		{
			name:                 "No file",
			givenPrefix:          "/images",
			givenRoot:            "_fixture/scripts",
			whenURL:              "/test/images/bolt.png",
			expectStatus:         http.StatusNotFound,
			expectBodyStartsWith: "{\"message\":\"Not Found\"}\n",
		},
		{
			name:                 "Directory",
			givenPrefix:          "/images",
			givenRoot:            "_fixture/images",
			whenURL:              "/test/images/",
			expectStatus:         http.StatusNotFound,
			expectBodyStartsWith: "{\"message\":\"Not Found\"}\n",
		},
		{
			name:                 "Directory Redirect",
			givenPrefix:          "/",
			givenRoot:            "_fixture",
			whenURL:              "/test/folder",
			expectStatus:         http.StatusMovedPermanently,
			expectHeaderLocation: "/test/folder/",
			expectBodyStartsWith: "",
		},
		{
			name:                 "Directory Redirect with non-root path",
			givenPrefix:          "/static",
			givenRoot:            "_fixture",
			whenURL:              "/test/static",
			expectStatus:         http.StatusMovedPermanently,
			expectHeaderLocation: "/test/static/",
			expectBodyStartsWith: "",
		},
		{
			name:                 "Prefixed directory 404 (request URL without slash)",
			givenPrefix:          "/folder/", // trailing slash will intentionally not match "/folder"
			givenRoot:            "_fixture",
			whenURL:              "/test/folder", // no trailing slash
			expectStatus:         http.StatusNotFound,
			expectBodyStartsWith: "{\"message\":\"Not Found\"}\n",
		},
		{
			name:                 "Prefixed directory redirect (without slash redirect to slash)",
			givenPrefix:          "/folder", // no trailing slash shall match /folder and /folder/*
			givenRoot:            "_fixture",
			whenURL:              "/test/folder", // no trailing slash
			expectStatus:         http.StatusMovedPermanently,
			expectHeaderLocation: "/test/folder/",
			expectBodyStartsWith: "",
		},
		{
			name:                 "Directory with index.html",
			givenPrefix:          "/",
			givenRoot:            "_fixture",
			whenURL:              "/test/",
			expectStatus:         http.StatusOK,
			expectBodyStartsWith: "<!doctype html>",
		},
		{
			name:                 "Prefixed directory with index.html (prefix ending with slash)",
			givenPrefix:          "/assets/",
			givenRoot:            "_fixture",
			whenURL:              "/test/assets/",
			expectStatus:         http.StatusOK,
			expectBodyStartsWith: "<!doctype html>",
		},
		{
			name:                 "Prefixed directory with index.html (prefix ending without slash)",
			givenPrefix:          "/assets",
			givenRoot:            "_fixture",
			whenURL:              "/test/assets/",
			expectStatus:         http.StatusOK,
			expectBodyStartsWith: "<!doctype html>",
		},
		{
			name:                 "Sub-directory with index.html",
			givenPrefix:          "/",
			givenRoot:            "_fixture",
			whenURL:              "/test/folder/",
			expectStatus:         http.StatusOK,
			expectBodyStartsWith: "<!doctype html>",
		},
		{
			name:                  "nok, URL encoded path traversal (single encoding, slash - unix separator)",
			givenRoot:             "_fixture/dist/public",
			whenURL:               "/%2e%2e%2fprivate.txt",
			expectStatus:          http.StatusNotFound,
			expectBodyStartsWith:  "{\"message\":\"Not Found\"}\n",
			expectBodyNotContains: `private file`,
		},
		{
			name:                  "nok, URL encoded path traversal (single encoding, backslash - windows separator)",
			givenRoot:             "_fixture/dist/public",
			whenURL:               "/%2e%2e%5cprivate.txt",
			expectStatus:          http.StatusNotFound,
			expectBodyStartsWith:  "{\"message\":\"Not Found\"}\n",
			expectBodyNotContains: `private file`,
		},
		{
			name:                 "do not allow directory traversal (backslash - windows separator)",
			givenPrefix:          "/",
			givenRoot:            "_fixture/",
			whenURL:              `/test/..\\middleware/basic_auth.go`,
			expectStatus:         http.StatusNotFound,
			expectBodyStartsWith: "{\"message\":\"Not Found\"}\n",
		},
		{
			name:                 "do not allow directory traversal (slash - unix separator)",
			givenPrefix:          "/",
			givenRoot:            "_fixture/",
			whenURL:              `/test/../middleware/basic_auth.go`,
			expectStatus:         http.StatusNotFound,
			expectBodyStartsWith: "{\"message\":\"Not Found\"}\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := New()

			g := e.Group("/test")
			g.Static(tc.givenPrefix, tc.givenRoot)

			req := httptest.NewRequest(http.MethodGet, tc.whenURL, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectStatus, rec.Code)
			body := rec.Body.String()
			if tc.expectBodyStartsWith != "" {
				assert.True(t, strings.HasPrefix(body, tc.expectBodyStartsWith))
			} else {
				assert.Equal(t, "", body)
			}
			if tc.expectBodyNotContains != "" {
				assert.NotContains(t, body, tc.expectBodyNotContains)
			}

			if tc.expectHeaderLocation != "" {
				assert.Equal(t, tc.expectHeaderLocation, rec.Result().Header["Location"][0])
			} else {
				_, ok := rec.Result().Header["Location"]
				assert.False(t, ok)
			}
		})
	}
}

func TestGroup_FileFS(t *testing.T) {
	var testCases = []struct {
		whenFS           fs.FS
		name             string
		whenPath         string
		whenFile         string
		givenURL         string
		expectStartsWith []byte
		expectCode       int
	}{
		{
			name:             "ok",
			whenPath:         "/walle",
			whenFS:           os.DirFS("_fixture/images"),
			whenFile:         "walle.png",
			givenURL:         "/assets/walle",
			expectCode:       http.StatusOK,
			expectStartsWith: []byte{0x89, 0x50, 0x4e},
		},
		{
			name:             "nok, requesting invalid path",
			whenPath:         "/walle",
			whenFS:           os.DirFS("_fixture/images"),
			whenFile:         "walle.png",
			givenURL:         "/assets/walle.png",
			expectCode:       http.StatusNotFound,
			expectStartsWith: []byte(`{"message":"Not Found"}`),
		},
		{
			name:             "nok, serving not existent file from filesystem",
			whenPath:         "/walle",
			whenFS:           os.DirFS("_fixture/images"),
			whenFile:         "not-existent.png",
			givenURL:         "/assets/walle",
			expectCode:       http.StatusNotFound,
			expectStartsWith: []byte(`{"message":"Not Found"}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := New()
			g := e.Group("/assets")
			g.FileFS(tc.whenPath, tc.whenFile, tc.whenFS)

			req := httptest.NewRequest(http.MethodGet, tc.givenURL, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectCode, rec.Code)

			body := rec.Body.Bytes()
			if len(body) > len(tc.expectStartsWith) {
				body = body[:len(tc.expectStartsWith)]
			}
			assert.Equal(t, tc.expectStartsWith, body)
		})
	}
}

func TestGroup_StaticPanic(t *testing.T) {
	var testCases = []struct {
		name      string
		givenRoot string
	}{
		{
			name:      "panics for ../",
			givenRoot: "../images",
		},
		{
			name:      "panics for /",
			givenRoot: "/images",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := New()
			e.Filesystem = os.DirFS("./")

			g := e.Group("/assets")

			assert.Panics(t, func() {
				g.Static("/images", tc.givenRoot)
			})
		})
	}
}

func TestGroup_RouteNotFoundWithMiddleware(t *testing.T) {
	var testCases = []struct {
		expectBody             any
		name                   string
		whenURL                string
		expectCode             int
		givenCustom404         bool
		expectMiddlewareCalled bool
	}{
		{
			name:                   "ok, custom 404 handler is called with middleware",
			givenCustom404:         true,
			whenURL:                "/group/test3",
			expectBody:             "404 GET /group/*",
			expectCode:             http.StatusNotFound,
			expectMiddlewareCalled: true, // because RouteNotFound is added after middleware is added
		},
		{
			name:                   "ok, default group 404 handler is not called with middleware",
			givenCustom404:         false,
			whenURL:                "/group/test3",
			expectBody:             "404 GET /*",
			expectCode:             http.StatusNotFound,
			expectMiddlewareCalled: false, // because RouteNotFound is added before middleware is added
		},
		{
			name:                   "ok, (no slash) default group 404 handler is called with middleware",
			givenCustom404:         false,
			whenURL:                "/group",
			expectBody:             "404 GET /*",
			expectCode:             http.StatusNotFound,
			expectMiddlewareCalled: false, // because RouteNotFound is added before middleware is added
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			okHandler := func(c *Context) error {
				return c.String(http.StatusOK, c.Request().Method+" "+c.Path())
			}
			notFoundHandler := func(c *Context) error {
				return c.String(http.StatusNotFound, "404 "+c.Request().Method+" "+c.Path())
			}

			e := New()
			e.GET("/test1", okHandler)
			e.RouteNotFound("/*", notFoundHandler)

			g := e.Group("/group")
			g.GET("/test1", okHandler)

			middlewareCalled := false
			g.Use(func(next HandlerFunc) HandlerFunc {
				return func(c *Context) error {
					middlewareCalled = true
					return next(c)
				}
			})
			if tc.givenCustom404 {
				g.RouteNotFound("/*", notFoundHandler)
			}

			req := httptest.NewRequest(http.MethodGet, tc.whenURL, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectMiddlewareCalled, middlewareCalled)
			assert.Equal(t, tc.expectCode, rec.Code)
			assert.Equal(t, tc.expectBody, rec.Body.String())
		})
	}
}

func TestGroup_GET(t *testing.T) {
	e := New()

	users := e.Group("/users")
	ri := users.GET("/list", func(c *Context) error {
		return c.String(http.StatusTeapot, "OK")
	})

	assert.Equal(t, http.MethodGet, ri.Method)
	assert.Equal(t, "/users/list", ri.Path)
	assert.Equal(t, http.MethodGet+":/users/list", ri.Name)
	assert.Nil(t, ri.Parameters)

	status, body := request(http.MethodGet, "/users/list", e)
	assert.Equal(t, http.StatusTeapot, status)
	assert.Equal(t, `OK`, body)
}

func TestGroup_Add(t *testing.T) {
	e := New()
	g := e.Group("/api")

	ri := g.Add(http.MethodPost, "/items", func(c *Context) error {
		return c.String(http.StatusCreated, "created")
	})

	assert.Equal(t, http.MethodPost, ri.Method)
	assert.Equal(t, "/api/items", ri.Path)

	status, body := request(http.MethodPost, "/api/items", e)
	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "created", body)
}

func TestGroup_AddPanicsOnDuplicate(t *testing.T) {
	e := New()
	g := e.Group("/api")
	h := func(c *Context) error { return nil }

	g.GET("/dup", h)

	assert.Panics(t, func() {
		g.GET("/dup", h)
	})
}

func TestGroup_AddRoute(t *testing.T) {
	e := New()
	g := e.Group("/api")

	ri, err := g.AddRoute(Route{
		Method:  http.MethodPut,
		Path:    "/items",
		Handler: func(c *Context) error { return c.String(http.StatusOK, "updated") },
	})
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPut, ri.Method)
	assert.Equal(t, "/api/items", ri.Path)

	status, body := request(http.MethodPut, "/api/items", e)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "updated", body)
}

func TestGroup_AddRouteReturnsErrorOnDuplicate(t *testing.T) {
	e := New()
	g := e.Group("/api")

	h := func(c *Context) error { return nil }
	_, err := g.AddRoute(Route{Method: http.MethodGet, Path: "/x", Handler: h})
	assert.NoError(t, err)

	_, err = g.AddRoute(Route{Method: http.MethodGet, Path: "/x", Handler: h})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate route")
}

func TestGroup_AddRouteAppliesGroupMiddleware(t *testing.T) {
	e := New()

	mwCalled := false
	g := e.Group("/api", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			mwCalled = true
			return next(c)
		}
	})

	_, err := g.AddRoute(Route{
		Method:  http.MethodGet,
		Path:    "/check",
		Handler: func(c *Context) error { return c.String(http.StatusOK, "ok") },
	})
	assert.NoError(t, err)

	status, _ := request(http.MethodGet, "/api/check", e)
	assert.Equal(t, http.StatusOK, status)
	assert.True(t, mwCalled)
}

func TestGroup_MatchEmptyMethods(t *testing.T) {
	e := New()
	g := e.Group("/api")

	ris := g.Match([]string{}, "/items", func(c *Context) error {
		return c.String(http.StatusOK, "ok")
	})
	assert.Empty(t, ris)
}

func TestGroup_AnyRespondsToAllMethods(t *testing.T) {
	e := New()
	g := e.Group("/api")
	g.Any("/resource", func(c *Context) error {
		return c.String(http.StatusOK, c.Request().Method)
	})

	for _, method := range []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace,
		http.MethodConnect,
	} {
		status, body := request(method, "/api/resource", e)
		assert.Equal(t, http.StatusOK, status, "method: %s", method)
		assert.Equal(t, method, body, "method: %s", method)
	}
}

func TestGroup_NestedGroupMiddlewareInheritance(t *testing.T) {
	e := New()

	var order []string

	mwA := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "A")
			return next(c)
		}
	}
	mwB := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "B")
			return next(c)
		}
	}
	mwC := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			order = append(order, "C")
			return next(c)
		}
	}

	api := e.Group("/api", mwA)
	v1 := api.Group("/v1", mwB)
	v1.GET("/hello", func(c *Context) error {
		order = append(order, "handler")
		return c.String(http.StatusOK, "hi")
	}, mwC)

	status, body := request(http.MethodGet, "/api/v1/hello", e)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "hi", body)
	// Middleware should execute in order: group A, group B, route-level C, then handler
	assert.Equal(t, []string{"A", "B", "C", "handler"}, order)
}

func TestGroup_NestedGroupPrefix(t *testing.T) {
	e := New()

	a := e.Group("/a")
	b := a.Group("/b")
	c := b.Group("/c")
	c.GET("/d", func(c *Context) error {
		return c.String(http.StatusOK, "deep")
	})

	status, body := request(http.MethodGet, "/a/b/c/d", e)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "deep", body)
}

func TestGroup_StaticFS(t *testing.T) {
	e := New()
	g := e.Group("/assets")

	ri := g.StaticFS("/files", os.DirFS("_fixture"))
	assert.Equal(t, http.MethodGet, ri.Method)
	assert.Equal(t, "/assets/files*", ri.Path)

	req := httptest.NewRequest(http.MethodGet, "/assets/files/index.html", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Body.String(), "<!doctype html>"))
}

func TestGroup_FileRouteInfo(t *testing.T) {
	e := New()
	g := e.Group("/dl")

	ri := g.File("/logo", "_fixture/images/walle.png")
	assert.Equal(t, http.MethodGet, ri.Method)
	assert.Equal(t, "/dl/logo", ri.Path)
	assert.Equal(t, "GET:/dl/logo", ri.Name)

	req := httptest.NewRequest(http.MethodGet, "/dl/logo", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGroup_FileFSRouteInfo(t *testing.T) {
	e := New()
	g := e.Group("/dl")

	ri := g.FileFS("/logo", "walle.png", os.DirFS("_fixture/images"))
	assert.Equal(t, http.MethodGet, ri.Method)
	assert.Equal(t, "/dl/logo", ri.Path)

	req := httptest.NewRequest(http.MethodGet, "/dl/logo", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGroup_UseAddsMiddlewareToGroup(t *testing.T) {
	e := New()
	g := e.Group("/api")

	called := false
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			called = true
			return next(c)
		}
	})
	g.GET("/ping", func(c *Context) error {
		return c.String(http.StatusOK, "pong")
	})

	status, body := request(http.MethodGet, "/api/ping", e)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "pong", body)
	assert.True(t, called)
}

func TestGroup_HTTPMethodShortcutsPrefixApplied(t *testing.T) {
	// Verify all HTTP method shortcuts apply the group prefix correctly.
	methods := []struct {
		name   string
		method string
		reg    func(g *Group, path string, h HandlerFunc) RouteInfo
	}{
		{"CONNECT", http.MethodConnect, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.CONNECT(p, h) }},
		{"DELETE", http.MethodDelete, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.DELETE(p, h) }},
		{"GET", http.MethodGet, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.GET(p, h) }},
		{"HEAD", http.MethodHead, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.HEAD(p, h) }},
		{"OPTIONS", http.MethodOptions, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.OPTIONS(p, h) }},
		{"PATCH", http.MethodPatch, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.PATCH(p, h) }},
		{"POST", http.MethodPost, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.POST(p, h) }},
		{"PUT", http.MethodPut, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.PUT(p, h) }},
		{"TRACE", http.MethodTrace, func(g *Group, p string, h HandlerFunc) RouteInfo { return g.TRACE(p, h) }},
	}

	for _, tc := range methods {
		t.Run(tc.name, func(t *testing.T) {
			e := New()
			g := e.Group("/pfx")
			h := func(c *Context) error { return c.String(http.StatusOK, tc.name) }

			ri := tc.reg(g, "/test", h)
			assert.Equal(t, tc.method, ri.Method)
			assert.Equal(t, "/pfx/test", ri.Path)
		})
	}
}

func TestGroup_RouteNotFoundPrefixApplied(t *testing.T) {
	e := New()
	g := e.Group("/api")

	ri := g.RouteNotFound("/*", func(c *Context) error {
		return c.NoContent(http.StatusNotFound)
	})
	assert.Equal(t, RouteNotFound, ri.Method)
	assert.Equal(t, "/api/*", ri.Path)
}

func TestGroup_AddRouteWithRouteMiddleware(t *testing.T) {
	e := New()

	groupMW := false
	routeMW := false

	g := e.Group("/api", func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			groupMW = true
			return next(c)
		}
	})

	_, err := g.AddRoute(Route{
		Method:  http.MethodGet,
		Path:    "/test",
		Handler: func(c *Context) error { return c.String(http.StatusOK, "ok") },
		Middlewares: []MiddlewareFunc{
			func(next HandlerFunc) HandlerFunc {
				return func(c *Context) error {
					routeMW = true
					return next(c)
				}
			},
		},
	})
	assert.NoError(t, err)

	status, _ := request(http.MethodGet, "/api/test", e)
	assert.Equal(t, http.StatusOK, status)
	assert.True(t, groupMW, "group middleware should be called")
	assert.True(t, routeMW, "route middleware should be called")
}
