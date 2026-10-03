package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

type validation struct {
	Errors   []string
	Warnings []string
}
type schemaValidator struct {
	root           map[string]any
	unknownIsError bool
}

func validateJSON(data []byte, schemaName string, unknownIsError bool) (validation, []byte, error) {
	value, err := decodeSchemaJSON(data)
	if err != nil {
		return validation{}, nil, invalid("Invalid JSON: %v", err)
	}
	schemaData, err := resources.ReadFile("schema/" + schemaName)
	if err != nil {
		return validation{}, nil, err
	}
	schemaValue, err := decodeSchemaJSON(schemaData)
	if err != nil {
		return validation{}, nil, err
	}
	schema := schemaValue.(map[string]any)
	v := schemaValidator{schema, unknownIsError}
	normalized, err := json.Marshal(value)
	return v.check(value, schema, "(root)"), normalized, err
}

func decodeSchemaJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("multiple JSON values")
	}
	return normalizeSchemaNumbers(value), nil
}

func normalizeSchemaNumbers(value any) any {
	switch value := value.(type) {
	case json.Number:
		if integer, err := value.Int64(); err == nil {
			return integer
		}
		if exact, ok := new(big.Rat).SetString(string(value)); ok && exact.IsInt() && exact.Num().IsInt64() {
			return exact.Num().Int64()
		}
	case map[string]any:
		for key, entry := range value {
			value[key] = normalizeSchemaNumbers(entry)
		}
	case []any:
		for i, entry := range value {
			value[i] = normalizeSchemaNumbers(entry)
		}
	}
	return value
}

func (v schemaValidator) check(value any, schema map[string]any, path string) validation {
	result := validation{Errors: []string{}, Warnings: []string{}}
	fail := func(message string) { result.Errors = append(result.Errors, path+": "+message) }
	merge := func(other validation) {
		result.Errors = append(result.Errors, other.Errors...)
		result.Warnings = append(result.Warnings, other.Warnings...)
	}
	if ref, ok := schema["$ref"].(string); ok {
		if ref == "#/$defs/relPath" {
			if s, ok := value.(string); !ok || !relativeFile(s) {
				fail("expected a relative path with forward slashes and no '..'")
			}
			return result
		}
		definition, ok := v.root["$defs"].(map[string]any)[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
		if !ok {
			fail("unsupported schema reference " + ref)
			return result
		}
		return v.check(value, definition, path)
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if alternatives, ok := schema[key].([]any); ok {
			matches := 0
			var best validation
			for i, alternative := range alternatives {
				candidate := v.check(value, alternative.(map[string]any), path)
				if len(candidate.Errors) == 0 {
					matches++
					best = candidate
				} else if matches == 0 && (i == 0 || len(candidate.Errors) < len(best.Errors)) {
					best = candidate
				}
			}
			if key == "oneOf" && matches > 1 {
				fail("matches more than one schema alternative")
			} else {
				merge(best)
			}
			return result
		}
	}
	if expected, ok := schema["const"]; ok && !reflect.DeepEqual(value, expected) {
		fail(fmt.Sprintf("expected %v", expected))
	}
	if allowed, ok := schema["enum"].([]any); ok && !slices.ContainsFunc(allowed, func(a any) bool { return reflect.DeepEqual(a, value) }) {
		fail(fmt.Sprintf("expected one of %v", allowed))
	}
	if kind, ok := schema["type"].(string); ok {
		valid := false
		switch kind {
		case "null":
			valid = value == nil
		case "string":
			_, valid = value.(string)
		case "object":
			_, valid = value.(map[string]any)
		case "array":
			_, valid = value.([]any)
		case "boolean":
			_, valid = value.(bool)
		case "integer":
			_, valid = value.(int64)
		}
		if !valid {
			fail("expected " + kind)
			return result
		}
	}
	switch value := value.(type) {
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		if required, ok := schema["required"].([]any); ok {
			for _, key := range required {
				if _, ok := value[key.(string)]; !ok {
					result.Errors = append(result.Errors, path+"."+key.(string)+": is required")
				}
			}
		}
		keys := slices.Sorted(maps.Keys(value))
		for _, key := range keys {
			if property, ok := properties[key].(map[string]any); ok {
				merge(v.check(value[key], property, path+"."+key))
			} else if schema["additionalProperties"] == false {
				message := path + "." + key + ": unknown field"
				alias := false
				for name := range properties {
					if strings.EqualFold(name, key) {
						alias = true
						break
					}
				}
				if v.unknownIsError || alias {
					result.Errors = append(result.Errors, message)
				} else {
					result.Warnings = append(result.Warnings, message)
				}
			}
		}
	case []any:
		if minimum, ok := schema["minItems"].(int64); ok && int64(len(value)) < minimum {
			fail(fmt.Sprintf("expected at least %d items", int(minimum)))
		}
		if item, ok := schema["items"].(map[string]any); ok {
			for i, entry := range value {
				merge(v.check(entry, item, fmt.Sprintf("%s[%d]", path, i)))
			}
		}
	case string:
		if minimum, ok := schema["minLength"].(int64); ok && int64(len(strings.TrimSpace(value))) < minimum {
			fail("must not be empty")
		}
		if pattern, ok := schema["pattern"].(string); ok {
			var matches bool
			if strings.Contains(pattern, "(?!") {
				matches = strings.TrimSpace(value) != "" && !filepath.IsAbs(value) && !strings.ContainsAny(value, "\\:")
			} else {
				re, err := regexp.Compile(pattern)
				if err != nil {
					fail("invalid embedded schema pattern")
					return result
				}
				matches = re.MatchString(value)
			}
			if !matches {
				fail("invalid value for pattern " + pattern)
			}
		}
		if schema["format"] == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
				fail("expected an ISO UTC timestamp")
			}
		}
	case int64:
		if minimum, ok := schema["minimum"].(int64); ok && value < minimum {
			fail(fmt.Sprintf("expected a value >= %v", minimum))
		}
	}
	return result
}
