package common

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func IsTrue(c types.Bool) bool {
	if c.IsNull() {
		return false
	}
	if c.IsUnknown() {
		return false
	}
	if c.ValueBool() {
		return true
	}
	return false
}

func GetString(c types.String) string {
	if c.IsNull() {
		return ""
	}
	if c.IsUnknown() {
		return ""
	}
	return c.ValueString()
}

func CopyPointer[T any](src *T) *T {
	if src == nil {
		return nil
	}
	dst := new(T)
	*dst = *src
	return dst
}
