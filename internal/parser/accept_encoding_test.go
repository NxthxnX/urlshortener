package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectAcceptEncoding(t *testing.T) {
	tests := []struct {
		name    string
		accept  []string
		support []string
		want    string
	}{
		{
			name:    "exact match",
			accept:  []string{"gzip"},
			support: []string{"gzip"},
			want:    "gzip",
		},
		{
			name:    "case insensitive",
			accept:  []string{"GZIP, Deflate"},
			support: []string{"gzip"},
			want:    "gzip",
		},
		{
			name:    "wildcard",
			accept:  []string{"*"},
			support: []string{"gzip"},
			want:    "gzip",
		},
		{
			name:    "q priority",
			accept:  []string{"gzip;q=0.5, deflate;q=0.8"},
			support: []string{"gzip", "deflate"},
			want:    "deflate",
		},
		{
			name:    "equal q prefers supported order",
			accept:  []string{"deflate, gzip;q=1"},
			support: []string{"gzip", "deflate"},
			want:    "gzip",
		},
		{
			name:    "q zero rejects encoding",
			accept:  []string{"gzip;q=0"},
			support: []string{"gzip"},
			want:    "identity",
		},
		{
			name:    "q zero with wildcard fallbacks",
			accept:  []string{"gzip;q=0, *;q=1"},
			support: []string{"gzip"},
			want:    "identity",
		},
		{
			name:    "empty header",
			accept:  []string{},
			support: []string{"gzip"},
			want:    "identity",
		},
		{
			name:    "identity q zero rejects identity",
			accept:  []string{"identity;q=0"},
			support: []string{""},
			want:    "",
		},
		{
			name:    "identity q zero does not block gzip",
			accept:  []string{"identity;q=0, gzip"},
			support: []string{"gzip"},
			want:    "gzip",
		},
		{
			name:    "wildcard q zero rejects all",
			accept:  []string{"*;q=0"},
			support: []string{"gzip"},
			want:    "",
		},
		{
			name:    "wildcard q zero with explicit gzip",
			accept:  []string{"*;q=0, gzip"},
			support: []string{"gzip"},
			want:    "gzip",
		},
		{
			name:    "unsupported encoding falls back to identity",
			accept:  []string{"deflate"},
			support: []string{"gzip"},
			want:    "identity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectAcceptEncoding(tt.accept, tt.support)
			assert.Equal(t, tt.want, got)
		})
	}
}
