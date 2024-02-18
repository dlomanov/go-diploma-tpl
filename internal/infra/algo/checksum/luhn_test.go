package checksum_test

import (
	"github.com/dlomanov/go-diploma-tpl/internal/infra/algo/checksum"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"1", "4242424242424242", true},
		{"2", "6200000000000005", true},
		{"3", "5534200028533164", true},
		{"4", "36227206271667", true},
		{"5", "471629309440", false},
		{"6", "1111", false},
		{"7", "12345674", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checksum.ValidateLuhn([]byte(tt.s))
			require.Equalf(t, tt.want, got, "ValidateLuhn() = %v, want %v", got, tt.want)
		})
	}
}
