package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type samplePatchRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func TestDecodeJSONPatchOmitVsNull(t *testing.T) {
	t.Run("omitted field is not present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(`{"name":"x"}`)))
		var body samplePatchRequest
		patch, err := decodeJSONPatch(req, &body, "name", "description")
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !patch.Has("name") {
			t.Fatal("expected name to be present")
		}
		if patch.Has("description") {
			t.Fatal("expected description to be omitted")
		}
		if body.Name == nil || *body.Name != "x" {
			t.Fatalf("unexpected name: %+v", body.Name)
		}
	})

	t.Run("null field is present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(`{"description":null}`)))
		var body samplePatchRequest
		patch, err := decodeJSONPatch(req, &body, "name", "description")
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !patch.Has("description") {
			t.Fatal("expected description to be present")
		}
		if !patch.IsNull("description") {
			t.Fatal("expected description to be null")
		}
		if body.Description != nil {
			t.Fatalf("expected nil description, got %+v", body.Description)
		}
	})

	t.Run("empty object is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(`{}`)))
		var body samplePatchRequest
		_, err := decodeJSONPatch(req, &body, "name", "description")
		if err != errNoFieldsToUpdate {
			t.Fatalf("expected errNoFieldsToUpdate, got %v", err)
		}
	})

	t.Run("unknown keys only is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte(`{"extra":"x"}`)))
		var body samplePatchRequest
		_, err := decodeJSONPatch(req, &body, "name", "description")
		if err != errNoFieldsToUpdate {
			t.Fatalf("expected errNoFieldsToUpdate, got %v", err)
		}
	})
}
