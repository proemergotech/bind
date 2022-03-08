package gentlemanbind

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/proemergotech/errors/v2"

	gctx "gopkg.in/h2non/gentleman.v2/context"
)

type handler struct {
	fn     gctx.Handler
	called bool
	err    error
}

func newHandler() *handler {
	h := &handler{}
	h.fn = gctx.NewHandler(func(c *gctx.Context) {
		h.err = c.Error
		h.called = true
	})
	return h
}

func TestEmpty(t *testing.T) {
	ctx := gctx.New()
	fn := newHandler()

	s := struct{}{}

	Bind(s).Exec("request", ctx, fn.fn)
	expect(t, fn.called, true)
	expect(t, ctx.Error, nil)
}

func TestNonStruct(t *testing.T) {
	cases := []struct {
		name    string
		request interface{}
	}{
		{
			name:    "string",
			request: "foo",
		},
		{
			name:    "int",
			request: 1,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := gctx.New()
			fn := newHandler()

			Bind(test.request).Exec("request", ctx, fn.fn)
			expect(t, fn.called, true)
			expect(t, fn.err.Error(), errors.New("input data must be a struct or a pointer to a struct").Error())
			expect(t, ctx.Request.Method, "GET")

			buf, err := ioutil.ReadAll(ctx.Request.Body)
			expect(t, err, nil)
			expect(t, ctx.Request.Header.Get("Content-Type"), "")
			expect(t, int(ctx.Request.ContentLength), 0)
			expect(t, string(buf), "")
			expect(t, ctx.Request.URL.Path, "")
			expect(t, ctx.Request.URL.RawQuery, "")
		})
	}
}

func TestURLParams(t *testing.T) {
	type subRequest struct {
		Param2 int `param:"param_2"`
	}
	cases := []struct {
		name     string
		request  interface{}
		pathIn   string
		wantPath string
	}{
		{
			name: "string_param",
			request: struct {
				Param1 string `param:"param_1"`
			}{
				Param1: "foo",
			},
			pathIn:   "/foo/:param_1",
			wantPath: "/foo/foo",
		},
		{
			name: "int_param",
			request: struct {
				Param1 int `param:"param_1"`
			}{
				Param1: 1,
			},
			pathIn:   "/foo/:param_1",
			wantPath: "/foo/1",
		},
		{
			name: "no_tag",
			request: struct {
				Param1 int
			}{
				Param1: 1,
			},
			pathIn:   "/foo/:param_1",
			wantPath: "/foo/:param_1",
		},
		{
			name:     "no_param",
			request:  struct{}{},
			pathIn:   "/foo/:param_1",
			wantPath: "/foo/:param_1",
		},
		{
			name: "anonym_param",
			request: struct {
				Param1 string `param:"param_1"`
				subRequest
			}{
				Param1: "foo",
				subRequest: subRequest{
					Param2: 1,
				},
			},
			pathIn:   "/foo/:param_1/bar/:param_2",
			wantPath: "/foo/foo/bar/1",
		},
		{
			name: "ptr_param",
			request: struct {
				Param1 *string `param:"param_1"`
			}{
				Param1: func() *string { s := "foo"; return &s }(),
			},
			pathIn:   "/foo/:param_1",
			wantPath: "/foo/foo",
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := gctx.New()
			ctx.Request.URL.Path = test.pathIn
			fn := newHandler()

			Bind(test.request).Exec("request", ctx, fn.fn)

			expect(t, fn.called, true)
			expect(t, ctx.Request.URL.Path, test.wantPath)
			expect(t, int(ctx.Request.ContentLength), 0)
		})
	}
}

func TestQueryParams(t *testing.T) {
	type subRequest struct {
		Param2 int `query:"param_2"`
	}
	cases := []struct {
		name      string
		request   interface{}
		wantQuery string
	}{
		{
			name: "string_query",
			request: struct {
				Param1 string `query:"param_1"`
			}{
				Param1: "foo",
			},
			wantQuery: "param_1=foo",
		},
		{
			name: "int_query",
			request: struct {
				Param1 int `query:"param_1"`
			}{
				Param1: 1,
			},
			wantQuery: "param_1=1",
		},
		{
			name: "slice_query",
			request: struct {
				Param1 []int `query:"param_1"`
			}{
				Param1: []int{1, 2},
			},
			wantQuery: "param_1=1&param_1=2",
		},
		{
			name: "no_tag",
			request: struct {
				Param1 int
			}{
				Param1: 1,
			},
			wantQuery: "",
		},
		{
			name:      "no_query",
			request:   struct{}{},
			wantQuery: "",
		},
		{
			name: "anonym_query",
			request: struct {
				Param1 string `query:"param_1"`
				subRequest
			}{
				Param1: "foo",
				subRequest: subRequest{
					Param2: 1,
				},
			},
			wantQuery: "param_1=foo&param_2=1",
		},
		{
			name: "string_ptr_query",
			request: struct {
				Param1 *string `query:"param_1"`
			}{
				Param1: func() *string { s := "foo"; return &s }(),
			},
			wantQuery: "param_1=foo",
		},
		{
			name: "slice_ptr_query",
			request: struct {
				Param1 *[]int `query:"param_1"`
			}{
				Param1: &[]int{1, 2},
			},
			wantQuery: "param_1=1&param_1=2",
		},
		{
			name: "multiple_query",
			request: struct {
				Param1 string `query:"param_1"`
				Param2 string `query:"param_2"`
			}{
				Param1: "foo",
				Param2: "bar",
			},
			wantQuery: "param_1=foo&param_2=bar",
		},
		{
			name: "omitempty",
			request: struct {
				Param1 string `query:"param_1,omitempty"`
				Param2 string `query:"param_2,omitempty"`
				Param3 string `query:"param_3"`
				Param4 string `query:"param_4"`
			}{
				Param1: "foo",
				Param2: "",
				Param3: "bar",
			},
			wantQuery: "param_1=foo&param_3=bar&param_4=",
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := gctx.New()
			fn := newHandler()

			Bind(test.request).Exec("request", ctx, fn.fn)

			expect(t, fn.called, true)
			expect(t, ctx.Request.URL.RawQuery, test.wantQuery)
			expect(t, int(ctx.Request.ContentLength), 0)
		})
	}
}

func TestBody(t *testing.T) {
	type subRequest struct {
		Param2 int `json:"param_2"`
	}
	type bar struct {
		String string `json:"bar_string"`
	}
	type Foo struct {
		Int  int `json:"foo_int"`
		Skip int
		Bars []*bar `json:"foo_bars"`
	}

	cases := []struct {
		name              string
		request           interface{}
		wantContentType   string
		wantContentLength int
		wantBody          string
	}{
		{
			name: "string_body",
			request: struct {
				Param1 string `json:"param_1"`
			}{
				Param1: "foo",
			},
			wantContentType:   "application/json",
			wantContentLength: 17,
			wantBody:          `{"param_1":"foo"}`,
		},
		{
			name: "int_body",
			request: struct {
				Param1 int `json:"param_1"`
			}{
				Param1: 1,
			},
			wantContentType:   "application/json",
			wantContentLength: 13,
			wantBody:          `{"param_1":1}`,
		},
		{
			name: "slice_body",
			request: struct {
				Param1 []int `json:"param_1"`
			}{
				Param1: []int{1, 2},
			},
			wantContentType:   "application/json",
			wantContentLength: 17,
			wantBody:          `{"param_1":[1,2]}`,
		},
		{
			name: "no_tag",
			request: struct {
				Param1 int
			}{
				Param1: 1,
			},
			wantContentType:   "",
			wantContentLength: 0,
			wantBody:          "",
		},
		{
			name:              "no_body",
			request:           struct{}{},
			wantContentType:   "",
			wantContentLength: 0,
			wantBody:          "",
		},
		{
			name: "anonym_body",
			request: struct {
				Param1 string `json:"param_1"`
				subRequest
			}{
				Param1: "foo",
				subRequest: subRequest{
					Param2: 1,
				},
			},
			wantContentType:   "application/json",
			wantContentLength: 29,
			wantBody:          `{"param_1":"foo","param_2":1}`,
		},
		{
			name: "string_ptr_body",
			request: struct {
				Param1 *string `json:"param_1"`
			}{
				Param1: func() *string { s := "foo"; return &s }(),
			},
			wantContentType:   "application/json",
			wantContentLength: 17,
			wantBody:          `{"param_1":"foo"}`,
		},
		{
			name: "map_body",
			request: struct {
				Param1 map[int]string `json:"param_1"`
			}{
				Param1: map[int]string{1: "foo", 2: "bar"},
			},
			wantContentType:   "application/json",
			wantContentLength: 33,
			wantBody:          `{"param_1":{"1":"foo","2":"bar"}}`,
		},
		{
			name: "complex_body",
			request: struct {
				String    string `json:"string"`
				Struct    Foo    `json:"struct"`
				StructPtr *Foo   `json:"struct_ptr"`
				Foo       `json:"embedded"`
				subRequest
			}{
				String: "foo",
				Struct: Foo{
					Int:  1,
					Skip: 2,
					Bars: []*bar{{String: "foo"}},
				},
				StructPtr: &Foo{
					Int:  3,
					Skip: 4,
					Bars: []*bar{{String: "foo"}},
				},
				Foo: Foo{
					Int:  5,
					Skip: 6,
					Bars: []*bar{{String: "foo"}},
				},
				subRequest: subRequest{
					Param2: 7,
				},
			},
			wantContentType:   "application/json",
			wantContentLength: 205,
			wantBody:          `{"string":"foo","struct":{"foo_int":1,"foo_bars":[{"bar_string":"foo"}]},"struct_ptr":{"foo_int":3,"foo_bars":[{"bar_string":"foo"}]},"embedded":{"foo_int":5,"foo_bars":[{"bar_string":"foo"}]},"param_2":7}`,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := gctx.New()
			fn := newHandler()

			Bind(test.request).Exec("request", ctx, fn.fn)

			buf, err := ioutil.ReadAll(ctx.Request.Body)
			expect(t, err, nil)
			expect(t, ctx.Request.Method, "GET")
			expect(t, fn.called, true)
			expect(t, ctx.Request.Header.Get("Content-Type"), test.wantContentType)
			expect(t, int(ctx.Request.ContentLength), test.wantContentLength)
			expect(t, string(buf), test.wantBody)
		})
	}
}

func TestMixed(t *testing.T) {
	type queryInner struct {
		Query2 []*int `query:"query_2"`
	}
	type bodyInner struct {
		Body3 string `json:"body_3"`
	}
	cases := []struct {
		name              string
		request           interface{}
		pathIn            string
		wantContentType   string
		wantContentLength int
		wantPath          string
		wantQuery         string
		wantBody          string
	}{
		{
			name: "simple_all_tags",
			request: struct {
				Param1 string `param:"param_1"`
				Query1 string `query:"query_1"`
				Body1  string `json:"body_1"`
				Skip   string
			}{
				Param1: "foo",
				Query1: "bar",
				Body1:  "baz",
			},
			pathIn:            "/foo/:param_1",
			wantContentType:   "application/json",
			wantContentLength: 16,
			wantPath:          "/foo/foo",
			wantQuery:         "query_1=bar",
			wantBody:          `{"body_1":"baz"}`,
		},
		{
			name: "complex_all_tags",
			request: struct {
				Param1 string `param:"param_1"`
				Param2 int    `param:"param_2"`
				Query1 string `query:"query_1"`
				queryInner
				Body1 *string `json:"body_1"`
				Body2 []int   `json:"body_2"`
				*bodyInner
			}{
				Param1: "foo",
				Param2: 1,
				Query1: "bar",
				queryInner: queryInner{[]*int{
					func() *int { i := 2; return &i }(),
					func() *int { i := 3; return &i }(),
				}},
				Body1:     func() *string { s := "baz"; return &s }(),
				Body2:     []int{4},
				bodyInner: &bodyInner{Body3: "fubar"},
			},
			pathIn:            "/foo/:param_1/bar/:param_2",
			wantContentType:   "application/json",
			wantContentLength: 46,
			wantPath:          "/foo/foo/bar/1",
			wantQuery:         "query_1=bar&query_2=2&query_2=3",
			wantBody:          `{"body_1":"baz","body_2":[4],"body_3":"fubar"}`,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := gctx.New()
			ctx.Request.URL.Path = test.pathIn
			fn := newHandler()

			Bind(test.request).Exec("request", ctx, fn.fn)

			buf, err := ioutil.ReadAll(ctx.Request.Body)
			expect(t, err, nil)
			expect(t, ctx.Request.Method, "GET")
			expect(t, fn.called, true)
			expect(t, ctx.Request.Header.Get("Content-Type"), test.wantContentType)
			expect(t, int(ctx.Request.ContentLength), test.wantContentLength)
			expect(t, string(buf), test.wantBody)
			expect(t, ctx.Request.URL.Path, test.wantPath)
			expect(t, ctx.Request.URL.RawQuery, test.wantQuery)
			expect(t, []byte(ctx.Request.URL.RawQuery), []byte(test.wantQuery))
		})
	}
}

func expect(t *testing.T, have, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave: (%T) %+v\nwant: (%T) %+v", have, have, want, want)
	}
}
