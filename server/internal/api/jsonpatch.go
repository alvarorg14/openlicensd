package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var errNoFieldsToUpdate = errors.New("no fields to update")

type jsonPatch struct {
	raw map[string]json.RawMessage
}

func (p *jsonPatch) Has(key string) bool {
	_, ok := p.raw[key]
	return ok
}

func (p *jsonPatch) IsNull(key string) bool {
	raw, ok := p.raw[key]
	if !ok {
		return false
	}
	return string(raw) == "null"
}

func decodeJSONPatch(r *http.Request, dest any, allowedKeys ...string) (*jsonPatch, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return nil, err
	}

	allowed := make(map[string]struct{}, len(allowedKeys))
	for _, key := range allowedKeys {
		allowed[key] = struct{}{}
	}

	known := 0
	for key := range raw {
		if _, ok := allowed[key]; ok {
			known++
		}
	}
	if known == 0 {
		return nil, errNoFieldsToUpdate
	}

	return &jsonPatch{raw: raw}, nil
}
