package errors

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

func ReasonForError(err error) int {
	switch t := err.(type) {
	case *httputils.JSONClientError:
		return t.Code
	}
	return -1
}

func IsNotFound(err error) bool {
	if ReasonForError(err) == http.StatusNotFound {
		return true
	}
	return false
}
