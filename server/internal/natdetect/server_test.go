package natdetect

import "testing"

func TestShouldBindAlternateAddressSkipsCoveredWildcard(t *testing.T) {
	tests := []struct {
		name    string
		primary string
		alt     string
		want    bool
	}{
		{
			name:    "wildcard covers loopback",
			primary: "0.0.0.0",
			alt:     "127.0.0.1",
			want:    false,
		},
		{
			name:    "same address",
			primary: "127.0.0.1",
			alt:     "127.0.0.1",
			want:    false,
		},
		{
			name:    "different concrete addresses",
			primary: "127.0.0.1",
			alt:     "127.0.0.2",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldBindAlternateAddress(tt.primary, tt.alt); got != tt.want {
				t.Fatalf("shouldBindAlternateAddress(%q, %q) = %v, want %v", tt.primary, tt.alt, got, tt.want)
			}
		})
	}
}
