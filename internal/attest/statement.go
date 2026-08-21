// Package attest wraps a direction hash in an in-toto Statement v1 with a
// versioned direction-record predicate, and verifies such statements against
// a work unit or bundle. Plain structs and encoding/json — no SDK. The
// envelope is unsigned in this version but shaped so DSSE can be layered
// later without touching any field (Festival Bundle SPEC §11).
package attest

import (
	"encoding/json"
	"errors"
	"regexp"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// StatementType is the in-toto Statement v1 type URI.
const StatementType = "https://in-toto.io/Statement/v1"

// ErrMalformedStatement marks a document that is not a valid Statement v1.
var ErrMalformedStatement = errors.New("malformed statement")

var hexRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Statement is the in-toto Statement v1 envelope.
type Statement struct {
	Type          string          `json:"_type"`
	Subject       []Subject       `json:"subject"`
	PredicateType string          `json:"predicateType"`
	Predicate     json.RawMessage `json:"predicate"`
}

// Subject names an artifact and its digests (bare lowercase hex).
type Subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

// Marshal renders s as indented JSON with a trailing newline. Map keys are
// sorted by encoding/json, so equal statements produce equal bytes.
func Marshal(s Statement) ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, errs.Wrap("marshal statement", err)
	}
	return append(b, '\n'), nil
}

// ParseStatement decodes and validates a Statement v1.
func ParseStatement(b []byte) (Statement, error) {
	var s Statement
	if err := json.Unmarshal(b, &s); err != nil {
		return Statement{}, errs.Wrap("statement: decode", errors.Join(ErrMalformedStatement, err))
	}
	switch {
	case s.Type != StatementType:
		return Statement{}, errs.Wrap("statement: _type "+s.Type, ErrMalformedStatement)
	case len(s.Subject) == 0:
		return Statement{}, errs.Wrap("statement: no subject", ErrMalformedStatement)
	case s.PredicateType == "":
		return Statement{}, errs.Wrap("statement: no predicateType", ErrMalformedStatement)
	case len(s.Predicate) == 0:
		return Statement{}, errs.Wrap("statement: no predicate", ErrMalformedStatement)
	}
	for _, sub := range s.Subject {
		if len(sub.Digest) == 0 {
			return Statement{}, errs.Wrap("statement: subject "+sub.Name+" has no digest", ErrMalformedStatement)
		}
		for alg, v := range sub.Digest {
			if !hexRE.MatchString(v) {
				return Statement{}, errs.Wrap("statement: subject "+sub.Name+" digest "+alg+" is not bare lowercase hex", ErrMalformedStatement)
			}
		}
	}
	return s, nil
}
