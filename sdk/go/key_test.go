package openlicensd

import "testing"

func TestNormalizeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "01234-56789-ABCDE-FGHJK-MNPQR",
			want:  "01234-56789-ABCDE-FGHJK-MNPQR",
		},
		{
			name:  "lowercase",
			input: "01234-56789-abcde-fghjk-mnpqr",
			want:  "01234-56789-ABCDE-FGHJK-MNPQR",
		},
		{
			name:  "no dashes",
			input: "0123456789ABCDEFGHJKMNPQR",
			want:  "01234-56789-ABCDE-FGHJK-MNPQR",
		},
		{
			name:  "ambiguous chars",
			input: "0123I-5678L-ABCDO-FGHJK-MNPQR",
			want:  "01231-56781-ABCD0-FGHJK-MNPQR",
		},
		{
			name:  "whitespace",
			input: "  0123456789ABCDEFGHJKMNPQR  ",
			want:  "01234-56789-ABCDE-FGHJK-MNPQR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeKey(tt.input); got != tt.want {
				t.Fatalf("NormalizeKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateKeyFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "valid", key: "01234-56789-ABCDE-FGHJK-MNPQR", valid: true},
		{name: "invalid length", key: "01234-56789", valid: false},
		{name: "invalid char", key: "01234-56789-ABCDE-FGHJK-MNPQ!", valid: false},
		{name: "valid without dashes", key: "0123456789ABCDEFGHJKMNPQR", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidateKeyFormat(tt.key); got != tt.valid {
				t.Fatalf("ValidateKeyFormat() = %v, want %v", got, tt.valid)
			}
		})
	}
}
