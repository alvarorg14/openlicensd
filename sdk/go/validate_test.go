package openlicensd

import (
	"testing"
	"time"
)

func TestTimeUntilExpiry(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "nil expiry",
			expiresAt: nil,
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "past expiry",
			expiresAt: &past,
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "future expiry",
			expiresAt: &future,
			wantMin:   50 * time.Minute,
			wantMax:   61 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ValidationResult{ExpiresAt: tt.expiresAt}
			got := result.TimeUntilExpiry()
			if got < tt.wantMin || got > tt.wantMax {
				t.Fatalf("TimeUntilExpiry() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestExpiresWithin(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	soon := now.Add(30 * time.Minute)
	later := now.Add(2 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		within    time.Duration
		want      bool
	}{
		{
			name:      "nil expiry",
			expiresAt: nil,
			within:    time.Hour,
			want:      false,
		},
		{
			name:      "already expired",
			expiresAt: &past,
			within:    time.Hour,
			want:      true,
		},
		{
			name:      "expires within window",
			expiresAt: &soon,
			within:    time.Hour,
			want:      true,
		},
		{
			name:      "expires beyond window",
			expiresAt: &later,
			within:    time.Hour,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ValidationResult{ExpiresAt: tt.expiresAt}
			if got := result.ExpiresWithin(tt.within); got != tt.want {
				t.Fatalf("ExpiresWithin(%v) = %v, want %v", tt.within, got, tt.want)
			}
		})
	}
}
