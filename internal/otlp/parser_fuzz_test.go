package otlp

import (
	"bytes"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte(validDocument))
	f.Add([]byte(`{"resourceSpans":[]}`))
	f.Add([]byte("{\"resourceSpans\":[]}\n{\"resourceSpans\":[]}\n"))
	f.Add([]byte(`{"resourceSpans":[`))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 1<<20 {
			t.Skip()
		}
		_, _ = Parse(bytes.NewReader(input))
	})
}
