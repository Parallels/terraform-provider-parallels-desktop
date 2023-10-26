package helpers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func AttributeTypes[T any](ctx context.Context) (map[string]attr.Type, error) {
	var t T
	val := reflect.ValueOf(t)
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%T has unsupported type: %s", t, typ)
	}

	attributeTypes := make(map[string]attr.Type)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue // Skip unexported fields.
		}
		tag := field.Tag.Get(`tfsdk`)
		if tag == "-" {
			continue // Skip explicitly excluded fields.
		}
		if tag == "" {
			return nil, fmt.Errorf(`%T needs a struct tag for "tfsdk" on %s`, t, field.Name)
		}

		if v, ok := val.Field(i).Interface().(attr.Value); ok {
			attributeTypes[tag] = v.Type(ctx)
		}
	}

	return attributeTypes, nil
}

func AttributeTypesMust[T any](ctx context.Context) map[string]attr.Type {
	return Must(AttributeTypes[T](ctx))
}

// Must is a generic implementation of the Go Must idiom [1, 2]. It panics if
// the provided error is non-nil and returns x otherwise.
//
// [1]: https://pkg.go.dev/text/template#Must
// [2]: https://pkg.go.dev/regexp#MustCompile
func Must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}
