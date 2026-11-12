package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
)

const smallRequestLimit = 4096

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	if strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, smallRequestLimit))
	if err != nil {
		return errors.New("request exceeds 4 KiB")
	}
	if err := requireFields(data, reflect.TypeOf(value).Elem()); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return errors.New("invalid JSON request")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request must contain one JSON object")
	}
	return nil
}
