package oidc

import "testing"

func TestSanitizePictureURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid https url",
			input: "https://lh3.googleusercontent.com/a/example",
			want:  "https://lh3.googleusercontent.com/a/example",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "http rejected",
			input: "http://example.com/photo.jpg",
			want:  "",
		},
		{
			name:  "too long",
			input: "https://example.com/" + string(make([]byte, 2048)),
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizePictureURL(tt.input); got != tt.want {
				t.Fatalf("sanitizePictureURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
