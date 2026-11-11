package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// requireFields enforces generated required JSON tags before zero-value decoding.
// This preserves the difference between an entered zero salary and a missing salary.
func requireFields(data json.RawMessage, kind reflect.Type) error {
	if string(data) == "null" {
		return errors.New("required values cannot be null")
	}
	switch kind.Kind() {
	case reflect.Pointer:
		return requireFields(data, kind.Elem())
	case reflect.Struct:
		return requireObjectFields(data, kind)
	case reflect.Slice:
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for _, item := range items {
			if err := requireFields(item, kind.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

func requireObjectFields(data json.RawMessage, kind reflect.Type) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	for index := 0; index < kind.NumField(); index++ {
		field := kind.Field(index)
		tag := field.Tag.Get("json")
		name, optional := parseJSONTag(tag)
		value, present := object[name]
		if !present && optional {
			continue
		}
		if !present {
			return errors.New("missing required field: " + name)
		}
		if err := requireFields(value, field.Type); err != nil {
			return err
		}
	}
	return nil
}

func parseJSONTag(tag string) (string, bool) {
	name, options, _ := strings.Cut(tag, ",")
	return name, strings.Contains(options, "omitempty")
}
