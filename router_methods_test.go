// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2015 LabStack LLC and Echo contributors

package echo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func dummyRouteMethod(method string) *routeMethod {
	return &routeMethod{
		RouteInfo: &RouteInfo{Method: method, Path: "/test"},
		handler:   func(c *Context) error { return nil },
	}
}

func TestRouteMethods_SetAndFind_StandardMethods(t *testing.T) {
	standardMethods := []string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}

	for _, method := range standardMethods {
		t.Run(method, func(t *testing.T) {
			rm := new(routeMethods)
			r := dummyRouteMethod(method)

			rm.set(method, r)

			got := rm.find(method, false)
			assert.Equal(t, r, got, "find should return the handler set for %s", method)

			// find without exact match should not fall back to any
			assert.Nil(t, rm.find("UNKNOWN_METHOD", false))
		})
	}
}

func TestRouteMethods_SetAndFind_ExtendedMethods(t *testing.T) {
	extendedMethods := []string{PROPFIND, REPORT}

	for _, method := range extendedMethods {
		t.Run(method, func(t *testing.T) {
			rm := new(routeMethods)
			r := dummyRouteMethod(method)

			rm.set(method, r)

			got := rm.find(method, false)
			assert.Equal(t, r, got, "find should return the handler set for %s", method)
		})
	}
}

func TestRouteMethods_SetAndFind_CustomMethods(t *testing.T) {
	customMethods := []string{"COPY", "LOCK", "MKCOL", "MOVE", "UNLOCK"}

	rm := new(routeMethods)
	for _, method := range customMethods {
		r := dummyRouteMethod(method)
		rm.set(method, r)
	}

	for _, method := range customMethods {
		got := rm.find(method, false)
		assert.NotNil(t, got, "find should return handler for custom method %s", method)
		assert.Equal(t, method, got.Method)
	}

	// Removing a custom method handler (setting nil handler)
	rm.set("COPY", &routeMethod{
		RouteInfo: &RouteInfo{Method: "COPY"},
		handler:   nil,
	})
	assert.Nil(t, rm.find("COPY", false), "COPY handler should be removed")
	// Other custom methods should still work
	assert.NotNil(t, rm.find("LOCK", false), "LOCK handler should still exist")
}

func TestRouteMethods_SetAndFind_RouteAny(t *testing.T) {
	rm := new(routeMethods)
	r := dummyRouteMethod(RouteAny)

	rm.set(RouteAny, r)

	got := rm.find(RouteAny, false)
	assert.Equal(t, r, got)

	// RouteAny should not be findable by a regular HTTP method name
	assert.Nil(t, rm.find(http.MethodGet, false))
}

func TestRouteMethods_Find_FallbackToAny(t *testing.T) {
	rm := new(routeMethods)

	// Set RouteAny handler
	anyHandler := dummyRouteMethod(RouteAny)
	rm.set(RouteAny, anyHandler)

	// Set a specific method handler
	getHandler := dummyRouteMethod(http.MethodGet)
	rm.set(http.MethodGet, getHandler)

	// When fallbackToAny is true and no specific handler: should return any handler
	got := rm.find(http.MethodPost, true)
	assert.Equal(t, anyHandler, got, "should fall back to RouteAny handler")

	// When fallbackToAny is true and specific handler exists: should return specific
	got = rm.find(http.MethodGet, true)
	assert.Equal(t, getHandler, got, "should prefer specific handler over RouteAny")

	// When fallbackToAny is false and no specific handler: should return nil
	got = rm.find(http.MethodPost, false)
	assert.Nil(t, got, "should not fall back when fallbackToAny is false")
}

func TestRouteMethods_SetAndFind_RouteNotFound(t *testing.T) {
	rm := new(routeMethods)
	r := dummyRouteMethod(RouteNotFound)

	rm.set(RouteNotFound, r)

	got := rm.find(RouteNotFound, false)
	assert.Equal(t, r, got)

	// RouteNotFound should not be findable by regular HTTP methods
	assert.Nil(t, rm.find(http.MethodGet, false))
}

func TestRouteMethods_RouteNotFound_NotCountedAsHandler(t *testing.T) {
	rm := new(routeMethods)
	r := dummyRouteMethod(RouteNotFound)

	rm.set(RouteNotFound, r)

	assert.False(t, rm.isHandler(), "RouteNotFound alone should not make isHandler return true")
	assert.NotNil(t, rm.notFoundHandler, "notFoundHandler should be set")
}

func TestRouteMethods_IsHandler(t *testing.T) {
	testCases := []struct {
		name           string
		setup          func(rm *routeMethods)
		expectIsHandler bool
	}{
		{
			name:            "empty routeMethods",
			setup:           func(rm *routeMethods) {},
			expectIsHandler: false,
		},
		{
			name: "only RouteNotFound",
			setup: func(rm *routeMethods) {
				rm.set(RouteNotFound, dummyRouteMethod(RouteNotFound))
			},
			expectIsHandler: false,
		},
		{
			name: "GET handler",
			setup: func(rm *routeMethods) {
				rm.set(http.MethodGet, dummyRouteMethod(http.MethodGet))
			},
			expectIsHandler: true,
		},
		{
			name: "POST handler",
			setup: func(rm *routeMethods) {
				rm.set(http.MethodPost, dummyRouteMethod(http.MethodPost))
			},
			expectIsHandler: true,
		},
		{
			name: "PROPFIND handler",
			setup: func(rm *routeMethods) {
				rm.set(PROPFIND, dummyRouteMethod(PROPFIND))
			},
			expectIsHandler: true,
		},
		{
			name: "REPORT handler",
			setup: func(rm *routeMethods) {
				rm.set(REPORT, dummyRouteMethod(REPORT))
			},
			expectIsHandler: true,
		},
		{
			name: "RouteAny handler",
			setup: func(rm *routeMethods) {
				rm.set(RouteAny, dummyRouteMethod(RouteAny))
			},
			expectIsHandler: true,
		},
		{
			name: "custom method handler",
			setup: func(rm *routeMethods) {
				rm.set("COPY", dummyRouteMethod("COPY"))
			},
			expectIsHandler: true,
		},
		{
			name: "RouteNotFound + GET handler",
			setup: func(rm *routeMethods) {
				rm.set(RouteNotFound, dummyRouteMethod(RouteNotFound))
				rm.set(http.MethodGet, dummyRouteMethod(http.MethodGet))
			},
			expectIsHandler: true,
		},
		{
			name: "RouteNotFound + RouteAny handler",
			setup: func(rm *routeMethods) {
				rm.set(RouteNotFound, dummyRouteMethod(RouteNotFound))
				rm.set(RouteAny, dummyRouteMethod(RouteAny))
			},
			expectIsHandler: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rm := new(routeMethods)
			tc.setup(rm)
			assert.Equal(t, tc.expectIsHandler, rm.isHandler())
		})
	}
}

func TestRouteMethods_AllowHeader_StandardMethods(t *testing.T) {
	rm := new(routeMethods)

	// Only OPTIONS should be listed when no handlers are registered
	// (OPTIONS is always included as a base)
	rm.updateAllowHeader()
	assert.Equal(t, http.MethodOptions, rm.allowHeader)

	// Add GET handler
	rm.set(http.MethodGet, dummyRouteMethod(http.MethodGet))
	assert.Equal(t, "OPTIONS, GET", rm.allowHeader)

	// Add POST handler
	rm.set(http.MethodPost, dummyRouteMethod(http.MethodPost))
	assert.Equal(t, "OPTIONS, GET, POST", rm.allowHeader)

	// Add PUT handler
	rm.set(http.MethodPut, dummyRouteMethod(http.MethodPut))
	assert.Equal(t, "OPTIONS, GET, POST, PUT", rm.allowHeader)
}

func TestRouteMethods_AllowHeader_AllStandardMethods(t *testing.T) {
	rm := new(routeMethods)

	methods := []string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}
	for _, m := range methods {
		rm.set(m, dummyRouteMethod(m))
	}
	rm.set(PROPFIND, dummyRouteMethod(PROPFIND))
	rm.set(REPORT, dummyRouteMethod(REPORT))

	// Expected order matches knownMethods order
	expected := "OPTIONS, CONNECT, DELETE, GET, HEAD, PATCH, POST, PROPFIND, PUT, TRACE, REPORT"
	assert.Equal(t, expected, rm.allowHeader)
}

func TestRouteMethods_AllowHeader_WithRouteAny(t *testing.T) {
	rm := new(routeMethods)

	// When RouteAny is set, all known methods should be listed in Allow header
	rm.set(RouteAny, dummyRouteMethod(RouteAny))

	// Should contain OPTIONS and all known methods (but NOT "echo_route_any" itself)
	assert.Contains(t, rm.allowHeader, http.MethodOptions)
	assert.Contains(t, rm.allowHeader, http.MethodGet)
	assert.Contains(t, rm.allowHeader, http.MethodPost)
	assert.Contains(t, rm.allowHeader, http.MethodPut)
	assert.Contains(t, rm.allowHeader, http.MethodDelete)
	assert.Contains(t, rm.allowHeader, http.MethodPatch)
	assert.Contains(t, rm.allowHeader, http.MethodHead)
	assert.Contains(t, rm.allowHeader, http.MethodConnect)
	assert.Contains(t, rm.allowHeader, http.MethodTrace)
	assert.Contains(t, rm.allowHeader, PROPFIND)
	assert.Contains(t, rm.allowHeader, REPORT)
	assert.NotContains(t, rm.allowHeader, RouteAny)
	assert.NotContains(t, rm.allowHeader, RouteNotFound)

	// All known methods should be present (OPTIONS as base + 10 others in loop = numMethodSlots total)
	parts := strings.Split(rm.allowHeader, ", ")
	assert.Equal(t, numMethodSlots, len(parts), "Allow header should have %d methods total", numMethodSlots)
}

func TestRouteMethods_AllowHeader_RouteNotFoundExcluded(t *testing.T) {
	rm := new(routeMethods)

	rm.set(RouteNotFound, dummyRouteMethod(RouteNotFound))

	// RouteNotFound should NOT trigger updateAllowHeader (early return in set).
	// So allowHeader remains its zero value (empty string).
	assert.Equal(t, "", rm.allowHeader)
}

func TestRouteMethods_AllowHeader_CustomMethods(t *testing.T) {
	rm := new(routeMethods)

	rm.set(http.MethodGet, dummyRouteMethod(http.MethodGet))
	rm.set("COPY", dummyRouteMethod("COPY"))
	rm.set("LOCK", dummyRouteMethod("LOCK"))
	rm.set("MKCOL", dummyRouteMethod("MKCOL"))

	// Custom methods should be appended after known methods, in sorted order
	parts := strings.Split(rm.allowHeader, ", ")
	assert.Equal(t, "OPTIONS", parts[0])
	assert.Equal(t, "GET", parts[1])
	// Custom methods should be sorted
	assert.Equal(t, "COPY", parts[2])
	assert.Equal(t, "LOCK", parts[3])
	assert.Equal(t, "MKCOL", parts[4])
}

func TestRouteMethods_AllowHeader_OrderPreserved(t *testing.T) {
	// Verify that known methods appear in the canonical order defined by knownMethods
	rm := new(routeMethods)

	// Add methods in reverse/alphabetical order to verify canonical ordering
	methodsToAdd := []string{
		http.MethodTrace,
		REPORT,
		http.MethodPut,
		PROPFIND,
		http.MethodPost,
		http.MethodPatch,
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodDelete,
		http.MethodConnect,
	}
	for _, m := range methodsToAdd {
		rm.set(m, dummyRouteMethod(m))
	}

	expected := "OPTIONS, CONNECT, DELETE, GET, HEAD, PATCH, POST, PROPFIND, PUT, TRACE, REPORT"
	assert.Equal(t, expected, rm.allowHeader)
}

func TestRouteMethods_Set_RouteNotFoundDoesNotCallUpdateAllowHeader(t *testing.T) {
	rm := new(routeMethods)

	// Set a GET handler first to establish an Allow header
	rm.set(http.MethodGet, dummyRouteMethod(http.MethodGet))
	allowBefore := rm.allowHeader
	assert.Equal(t, "OPTIONS, GET", allowBefore)

	// Setting RouteNotFound should NOT change the Allow header
	rm.set(RouteNotFound, dummyRouteMethod(RouteNotFound))
	assert.Equal(t, allowBefore, rm.allowHeader, "RouteNotFound should not modify Allow header")
}

func TestRouteMethods_MethodIndex_Consistency(t *testing.T) {
	// Verify that methodIndex is consistent with knownMethods
	assert.Equal(t, numMethodSlots, len(methodIndex))
	for i, method := range knownMethods {
		idx, ok := methodIndex[method]
		assert.True(t, ok, "method %s should be in methodIndex", method)
		assert.Equal(t, i, idx, "method %s should have index %d", method, i)
	}

	// Verify that RouteAny and RouteNotFound are NOT in methodIndex
	_, ok := methodIndex[RouteAny]
	assert.False(t, ok, "RouteAny should not be in methodIndex")
	_, ok = methodIndex[RouteNotFound]
	assert.False(t, ok, "RouteNotFound should not be in methodIndex")
}

func TestRouteMethods_Find_RouteAnyNotResolvedByHTTPMethod(t *testing.T) {
	rm := new(routeMethods)

	// Set RouteAny but no specific methods
	rm.set(RouteAny, dummyRouteMethod(RouteAny))

	// Direct find by RouteAny
	assert.NotNil(t, rm.find(RouteAny, false))

	// Find by HTTP method without fallback - should be nil (no specific handler)
	assert.Nil(t, rm.find(http.MethodGet, false))

	// Find by HTTP method with fallback - should return RouteAny handler
	assert.NotNil(t, rm.find(http.MethodGet, true))
}

func TestRouteMethods_SetRemoveCustomMethod(t *testing.T) {
	rm := new(routeMethods)

	// Add custom method
	rm.set("COPY", dummyRouteMethod("COPY"))
	assert.NotNil(t, rm.find("COPY", false))
	assert.True(t, rm.isHandler())

	// Remove custom method by setting with nil handler
	rm.set("COPY", &routeMethod{
		RouteInfo: &RouteInfo{Method: "COPY"},
		handler:   nil,
	})
	assert.Nil(t, rm.find("COPY", false))
	assert.False(t, rm.isHandler())
}

func TestRouteMethods_Integration_AllowHeaderViaRouter(t *testing.T) {
	e := New()
	r := e.router

	// Register multiple methods for same path
	r.Add(Route{Method: http.MethodGet, Path: "/api", Handler: handlerFunc})
	r.Add(Route{Method: http.MethodPost, Path: "/api", Handler: handlerFunc})
	r.Add(Route{Method: http.MethodPut, Path: "/api", Handler: handlerFunc})
	r.Add(Route{Method: PROPFIND, Path: "/api", Handler: handlerFunc})

	// Send a request with a method that is NOT registered
	req := httptest.NewRequest(http.MethodDelete, "/api", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := r.Route(c)
	err := h(c)

	assert.Equal(t, ErrMethodNotAllowed, err)
	allowHeader := c.Get(ContextKeyHeaderAllow).(string)
	parts := strings.Split(allowHeader, ", ")
	assert.ElementsMatch(t, []string{"OPTIONS", "GET", "POST", "PUT", "PROPFIND"}, parts)
}

func TestRouteMethods_Integration_RouteAnyFallbackViaRouter(t *testing.T) {
	e := New()
	r := e.router

	// Register specific method and RouteAny
	r.Add(Route{Method: http.MethodGet, Path: "/api", Handler: handlerFunc, Name: "get"})
	r.Add(Route{Method: RouteAny, Path: "/api", Handler: handlerHelper("handler", 42), Name: "any"})

	// GET should match the specific handler
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := r.Route(c)
	assert.NoError(t, h(c))
	assert.Equal(t, "/api", c.Path())

	// POST should fall back to RouteAny handler
	req = httptest.NewRequest(http.MethodPost, "/api", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = r.Route(c)
	assert.NoError(t, h(c))
	assert.Equal(t, 42, c.Get("handler"))
}

func TestRouteMethods_Integration_CustomMethodViaRouter(t *testing.T) {
	e := New()
	r := e.router

	r.Add(Route{Method: "COPY", Path: "/resource", Handler: handlerFunc})

	// Exact match for custom method
	req := httptest.NewRequest("COPY", "/resource", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := r.Route(c)
	assert.NoError(t, h(c))

	// Different method should get 405
	req = httptest.NewRequest("MOVE", "/resource", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	h = r.Route(c)
	err := h(c)
	assert.Equal(t, ErrMethodNotAllowed, err)
}
