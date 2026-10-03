package audit

import (
	"slices"
	"testing"
)

func TestSchemaAlternativeDiagnostics(t *testing.T) {
	for _, test := range []struct {
		name, value, schema string
		errors, warnings    []string
	}{
		{
			name:   "fewest errors when no alternative matches",
			value:  `{}`,
			schema: `{"oneOf":[{"type":"object","required":["first","second"]},{"type":"object","required":["third"]}]}`,
			errors: []string{"(root).third: is required"},
		},
		{
			name:   "first alternative wins an error tie",
			value:  `42`,
			schema: `{"oneOf":[{"type":"string"},{"type":"boolean"}]}`,
			errors: []string{"(root): expected string"},
		},
		{
			name:     "one match retains sorted warnings",
			value:    `{"alpha":"a","beta":"b","gamma":"c"}`,
			schema:   `{"oneOf":[{"type":"string"},{"type":"object","properties":{"beta":{"type":"string"}},"additionalProperties":false}]}`,
			warnings: []string{"(root).alpha: unknown field", "(root).gamma: unknown field"},
		},
		{
			name:   "multiple oneOf matches suppress candidate warnings",
			value:  `{"alpha":"a","beta":"b"}`,
			schema: `{"oneOf":[{"type":"object","properties":{"alpha":{"type":"string"}},"additionalProperties":false},{"type":"object","properties":{"beta":{"type":"string"}},"additionalProperties":false}]}`,
			errors: []string{"(root): matches more than one schema alternative"},
		},
		{
			name:     "multiple anyOf matches retain the last candidate warnings",
			value:    `{"alpha":"a","beta":"b","gamma":"c"}`,
			schema:   `{"anyOf":[{"type":"object","properties":{"alpha":{"type":"string"}},"additionalProperties":false},{"type":"object","properties":{"beta":{"type":"string"}},"additionalProperties":false}]}`,
			warnings: []string{"(root).alpha: unknown field", "(root).gamma: unknown field"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, err := decodeSchemaJSON([]byte(test.value))
			if err != nil {
				t.Fatal(err)
			}
			schemaValue, err := decodeSchemaJSON([]byte(test.schema))
			if err != nil {
				t.Fatal(err)
			}
			schema := schemaValue.(map[string]any)
			validator := schemaValidator{root: schema}
			got := validator.check(value, schema, "(root)")
			if !slices.Equal(got.Errors, test.errors) || !slices.Equal(got.Warnings, test.warnings) {
				t.Fatalf("errors = %v, warnings = %v; want errors = %v, warnings = %v", got.Errors, got.Warnings, test.errors, test.warnings)
			}
		})
	}
}
