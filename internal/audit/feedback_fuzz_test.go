package audit

import (
	"strings"
	"testing"
)

func FuzzFeedbackBoundary(f *testing.F) {
	for _, seed := range []string{
		`{"decision":{"type":"coa","coaId":"expire-one-hour"},"coaNotes":{},"note":"Keep this"}`,
		`{"decision":{"type":"none","reason":"Another plan"},"note":"\u0000Hello\r\n"}`,
		`{"decision":{"type":"none","reason":"\u0000"}}`,
		`{"coaNotes":{"__proto__":"Bad option"}}`,
		`{"note":null}`, `{}`, `null`,
		`{"Note":null}`, `{"COANotes":{"expire-one-hour":null}}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		content, err := decodeContent(data)
		if err != nil {
			return
		}
		texts := []string{content.Note}
		for _, note := range content.COANotes {
			texts = append(texts, note)
		}
		if decision := content.Decision; decision != nil {
			switch decision.Type {
			case chooseOption:
				if decision.COAID == "" {
					t.Fatal("accepted an empty option identity")
				}
			case chooseNone:
				if strings.TrimSpace(decision.Reason) == "" {
					t.Fatal("accepted none without a reason")
				}
				texts = append(texts, decision.Reason)
			default:
				t.Fatal("accepted an unknown decision kind")
			}
		}
		for _, text := range texts {
			for _, r := range text {
				if r < 32 && r != '\n' && r != '\t' || r == 127 {
					t.Fatal("accepted feedback contains a forbidden control character")
				}
			}
		}
	})
}
