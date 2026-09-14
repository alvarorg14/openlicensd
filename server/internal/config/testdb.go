package config

import (
	"os"
	"testing"
)

func TestDatabaseURL(t *testing.T) string {
	t.Helper()

	databaseURL := os.Getenv("OPENLICENSD_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("OPENLICENSD_TEST_DATABASE_URL not set")
	}
	return databaseURL
}
