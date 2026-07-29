package harbor

import (
	"strings"
	"testing"
)

func TestSanitizeRobotNamePart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"openlicensd", "openlicensd"},
		{"X4F9K", "x4f9k"},
		{"my-project", "my-project"},
		{"my_project.v1", "myprojectv1"},
		{"***", "robot"},
		{"", "robot"},
	}

	for _, tc := range tests {
		got := sanitizeRobotNamePart(tc.in)
		if got != tc.want {
			t.Fatalf("sanitizeRobotNamePart(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildRobotNameAllowsOnlyHyphens(t *testing.T) {
	t.Parallel()

	name := buildRobotName("Open-Licensd", "X4F9K")
	if name != strings.ToLower(name) {
		t.Fatalf("name=%q must be lowercase", name)
	}
	if !strings.HasPrefix(name, "open-licensd-x4f9k-") {
		t.Fatalf("name=%q want prefix open-licensd-x4f9k-", name)
	}

	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			t.Fatalf("name=%q contains disallowed character %q", name, r)
		}
	}
}
