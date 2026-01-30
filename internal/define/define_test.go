package define

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTrustedProxyIPs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", []string{}},
		{"single", "192.168.1.1", []string{"192.168.1.1"}},
		{"comma_separated", "192.168.1.1, 10.0.0.1 , 172.16.0.1", []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"}},
		{"with_spaces", "  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"drops_empty", "a,,b,,c", []string{"a", "b", "c"}},
		{"all_empty", ", , , ", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTrustedProxyIPs(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
