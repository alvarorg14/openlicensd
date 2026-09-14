package store

import (
	"context"
	"testing"
)

func TestLicenseMetadataColumn(t *testing.T) {
	st := openIntegrationStore(t)
	ctx := context.Background()

	var dataType, isNullable string
	err := st.pool.QueryRow(ctx, `
		SELECT data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'licenses'
		  AND column_name = 'metadata'
	`).Scan(&dataType, &isNullable)
	if err != nil {
		t.Fatalf("query licenses.metadata column: %v", err)
	}
	if dataType != "jsonb" {
		t.Fatalf("expected data_type jsonb, got %q", dataType)
	}
	if isNullable != "YES" {
		t.Fatalf("expected metadata to be nullable, got is_nullable=%q", isNullable)
	}
}
