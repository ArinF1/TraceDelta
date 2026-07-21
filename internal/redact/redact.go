// Package redact removes sensitive OTLP attributes before comparison evidence
// can be constructed.
package redact

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ArinF1/TraceDelta/internal/model"
)

// Options controls attribute keys added to TraceDelta's built-in deny rules.
type Options struct {
	AdditionalKeys []string
}

// Snapshot returns a deep-copied snapshot with denied attributes removed. Key
// matching is case-insensitive. The input snapshot is never mutated.
func Snapshot(input model.Snapshot, options Options) (model.Snapshot, error) {
	policy, err := newPolicy(options.AdditionalKeys)
	if err != nil {
		return model.Snapshot{}, err
	}

	result := model.Snapshot{Traces: make([]model.Trace, len(input.Traces))}
	for traceIndex, trace := range input.Traces {
		result.Traces[traceIndex].ID = trace.ID
		result.Traces[traceIndex].Spans = make([]model.Span, len(trace.Spans))
		for spanIndex, span := range trace.Spans {
			redacted := span
			redacted.Attributes = policy.attributes(span.Attributes)
			redacted.Resource = span.Resource
			redacted.Resource.Attributes = policy.attributes(span.Resource.Attributes)
			redacted.Scope = span.Scope
			redacted.Scope.Attributes = policy.attributes(span.Scope.Attributes)
			if policy.denied("service.name") {
				redacted.ServiceName = ""
			}
			result.Traces[traceIndex].Spans[spanIndex] = redacted
		}
	}
	return result, nil
}

type policy struct {
	additional map[string]struct{}
}

func newPolicy(additional []string) (policy, error) {
	keys := make(map[string]struct{}, len(additional))
	for index, key := range additional {
		normalized := normalizeKey(key)
		if normalized == "" {
			return policy{}, fmt.Errorf("additional redacted attribute key at index %d must not be empty", index)
		}
		keys[normalized] = struct{}{}
	}
	return policy{additional: keys}, nil
}

func (p policy) attributes(input model.Attributes) model.Attributes {
	if input == nil {
		return nil
	}
	result := make(model.Attributes, len(input))
	for key, value := range input {
		if p.denied(key) {
			continue
		}
		result[key] = p.value(value)
	}
	return result
}

func (p policy) value(input model.AttributeValue) model.AttributeValue {
	result := input
	if input.BytesValue != nil {
		result.BytesValue = append([]byte(nil), input.BytesValue...)
	}
	if input.ArrayValue != nil {
		result.ArrayValue = make([]model.AttributeValue, len(input.ArrayValue))
		for index, value := range input.ArrayValue {
			result.ArrayValue[index] = p.value(value)
		}
	}
	if input.KVListValue != nil {
		result.KVListValue = make([]model.AttributeKeyValue, 0, len(input.KVListValue))
		for _, entry := range input.KVListValue {
			if p.denied(entry.Key) {
				continue
			}
			result.KVListValue = append(result.KVListValue, model.AttributeKeyValue{
				Key:   entry.Key,
				Value: p.value(entry.Value),
			})
		}
	}
	return result
}

func (p policy) denied(key string) bool {
	normalized := normalizeKey(key)
	if _, exists := p.additional[normalized]; exists {
		return true
	}
	if _, exists := builtInExactKeys[normalized]; exists {
		return true
	}
	for _, prefix := range builtInPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	for _, segment := range keySegments(normalized) {
		if _, exists := sensitiveSegments[segment]; exists {
			return true
		}
		for _, fragment := range sensitiveFragments {
			if strings.Contains(segment, fragment) {
				return true
			}
		}
	}
	return false
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func keySegments(key string) []string {
	return strings.FieldsFunc(key, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsDigit(character)
	})
}

var builtInExactKeys = map[string]struct{}{
	"api-key":              {},
	"api.key":              {},
	"api_key":              {},
	"client.address":       {},
	"db.connection_string": {},
	"db.query.text":        {},
	"db.statement":         {},
	"error.message":        {},
	"exception.message":    {},
	"exception.stacktrace": {},
	"network.peer.address": {},
	"private-key":          {},
	"private.key":          {},
	"private_key":          {},
	"server.address":       {},
	"url.full":             {},
	"url.path":             {},
	"url.query":            {},
	"user_agent.original":  {},
	"x-api-key":            {},
}

var builtInPrefixes = []string{
	"enduser.",
	"account.",
	"contact.",
	"customer.",
	"device.",
	"http.request.header.",
	"http.response.header.",
	"person.",
	"session.",
	"user.",
}

var sensitiveSegments = map[string]struct{}{
	"apikey":        {},
	"authorization": {},
	"cookie":        {},
	"credential":    {},
	"credentials":   {},
	"email":         {},
	"passwd":        {},
	"password":      {},
	"phone":         {},
	"secret":        {},
	"ssn":           {},
	"token":         {},
}

var sensitiveFragments = []string{
	"apikey",
	"authorization",
	"cookie",
	"credential",
	"email",
	"password",
	"phone",
	"privatekey",
	"secret",
	"token",
}
