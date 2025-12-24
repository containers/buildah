package internal

import (
	"encoding/json"
	"fmt"

	digest "github.com/opencontainers/go-digest"
)

// InTotoStatement represents an in-toto statement used in attestations.
// See https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md
type InTotoStatement struct {
	Type          string          `json:"_type"`
	Subject       []InTotoSubject `json:"subject"`
	PredicateType string          `json:"predicateType"`
	Predicate     json.RawMessage `json:"predicate"`
}

// InTotoSubject represents a subject in an in-toto statement.
type InTotoSubject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

// ParseInTotoStatement parses an in-toto statement from JSON bytes.
func ParseInTotoStatement(data []byte) (*InTotoStatement, error) {
	var statement InTotoStatement
	if err := json.Unmarshal(data, &statement); err != nil {
		return nil, NewInvalidSignatureError(fmt.Sprintf("parsing in-toto statement: %v", err))
	}

	// Validate required fields
	if statement.Type == "" {
		return nil, NewInvalidSignatureError("in-toto statement missing _type field")
	}
	if statement.Type != "https://in-toto.io/Statement/v1" && statement.Type != "https://in-toto.io/Statement/v0.1" {
		return nil, NewInvalidSignatureError(fmt.Sprintf("unsupported in-toto statement type: %s", statement.Type))
	}
	if len(statement.Subject) == 0 {
		return nil, NewInvalidSignatureError("in-toto statement has no subjects")
	}

	return &statement, nil
}

// MatchesDigest returns true if any of the statement's subjects matches the given digest.
func (s *InTotoStatement) MatchesDigest(d digest.Digest) bool {
	digestAlgo := d.Algorithm().String()
	digestValue := d.Encoded()

	for _, subject := range s.Subject {
		if subjectDigest, ok := subject.Digest[digestAlgo]; ok {
			if subjectDigest == digestValue {
				return true
			}
		}
	}
	return false
}
