package gentlemanbind

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"gitlab.com/proemergotech/errors"

	jsoniter "github.com/json-iterator/go"
	gctx "gopkg.in/h2non/gentleman.v2/context"
	p "gopkg.in/h2non/gentleman.v2/plugin"
)

type filterFunc func(reflect.Value, []string) (bool, error)

// Bind binds a struct to a gentleman request using the tags of the struct.
// If you want to bind a non-struct object, use the builtin JSON, Param(s) and (Set|Add)Query methods.
//
// The following tags are valid:
//		- json: marks body fields
//		- param: marks url params
//		- query: marks query params
func Bind(requestData interface{}) p.Plugin {
	return p.NewRequestPlugin(func(c *gctx.Context, h gctx.Handler) {
		if err := bindParams(c, requestData); err != nil {
			h.Error(c, err)
			return
		}
		if err := bindQuery(c, requestData); err != nil {
			h.Error(c, err)
			return
		}
		if err := bindBody(c, requestData); err != nil {
			h.Error(c, err)
			return
		}

		h.Next(c)
	})
}

func bindBody(c *gctx.Context, data interface{}) error {
	bodyJSON := jsoniter.Config{
		SortMapKeys:            true,
		ValidateJsonRawMessage: true,
		OnlyTaggedField:        true,
		TagKey:                 "json",
	}.Froze()

	body, err := bodyJSON.Marshal(data)
	if err != nil {
		return err
	}

	if string(body) == "{}" {
		return nil
	}

	// the code below was copied from the official gentleman body.JSON middleware: https://github.com/h2non/gentleman/blob/master/plugins/body/body.go
	if c.Request.Method == "" {
		c.Request.Method = "POST"
	}
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")

	return nil
}

func bindQuery(c *gctx.Context, data interface{}) error {
	m, err := extract(data, "query", queryFilter)
	if err != nil {
		return err
	}

	if len(m) == 0 {
		return nil
	}

	query := c.Request.URL.Query()

	for k, v := range m {
		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				vi := v.Index(i)
				// if it's a collection of pointers, get the values they're pointing to instead
				if vi.Kind() == reflect.Ptr {
					vi = vi.Elem()
				}
				query.Add(k, fmt.Sprintf("%v", vi))
			}
		default:
			query.Set(k, fmt.Sprintf("%v", v))
		}
	}
	c.Request.URL.RawQuery = query.Encode()

	return nil
}

func bindParams(c *gctx.Context, data interface{}) error {
	m, err := extract(data, "param", paramFilter)
	if err != nil {
		return err
	}

	for k, v := range m {
		c.Request.URL.Path = strings.Replace(c.Request.URL.Path, ":"+k, fmt.Sprintf("%v", v), -1)
	}

	return nil
}

// Extracts the field values of a struct that have `tagName` tag.
// Disregards empty and "-" tags, otherwise the extraction rules are specified in the passed filter function.
func extract(data interface{}, tagName string, filter filterFunc) (map[string]reflect.Value, error) {
	if data == nil || tagName == "" {
		return nil, nil
	}

	v := reflect.ValueOf(data)
	// in case v is a pointer, take the value it points to
	v = reflect.Indirect(v)
	// if the reflect.Value returned by reflect.Indirect() is not valid, then v was a nil pointer
	if !v.IsValid() {
		return nil, errors.New("data cannot be nil pointer")
	}

	// disregard non-struct input data
	if v.Type().Kind() != reflect.Struct {
		return nil, errors.New("input data must be a struct or a pointer to a struct")
	}

	return process(v, tagName, filter)
}

func process(v reflect.Value, tagName string, filter filterFunc) (map[string]reflect.Value, error) {
	ret := map[string]reflect.Value{}

	for i := 0; i < v.NumField(); i++ {
		vfi := v.Field(i)
		vfi = reflect.Indirect(vfi)
		if !vfi.IsValid() {
			continue
		}

		tfi := v.Type().Field(i)
		if tfi.Anonymous {
			if vfi.Kind() == reflect.Struct {
				sub, err := process(vfi, tagName, filter)
				if err != nil {
					return nil, err
				}
				for sk, sv := range sub {
					ret[sk] = sv
				}
				continue
			} else {
				return nil, errors.New("anonymous field must be a struct or a pointer to a struct")
			}
		}

		tag := tfi.Tag.Get(tagName)
		tagParts := strings.Split(tag, ",")
		if tagParts[0] == "" {
			continue
		}

		if ok, err := filter(vfi, tagParts[1:]); err != nil {
			return nil, err
		} else if ok {
			ret[tagParts[0]] = vfi
		}

		continue
	}

	return ret, nil
}

func queryFilter(v reflect.Value, tagOpts []string) (bool, error) {
	var omitempty bool
	for _, o := range tagOpts {
		if o == "omitempty" {
			omitempty = true
		}
	}

	switch v.Kind() {
	case reflect.Struct, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr:
		return false, errors.New("field type can't be bound as a query parameter")
	default:
		if omitempty && isEmptyValue(v) {
			return false, nil
		}
		return true, nil
	}
}

func paramFilter(v reflect.Value, _ []string) (bool, error) {
	switch v.Kind() {
	case reflect.Map, reflect.Array, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr:
		return false, errors.New("field type can't be bound as an url parameter")
	default:
		return true, nil
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
