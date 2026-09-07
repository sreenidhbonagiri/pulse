package repository

import "testing"

func TestClampIncidentLimit(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{in: 0, want: defaultIncidentLimit},
		{in: -5, want: defaultIncidentLimit},
		{in: 10, want: 10},
		{in: maxIncidentLimit, want: maxIncidentLimit},
		{in: maxIncidentLimit + 1, want: maxIncidentLimit},
	}

	for _, test := range tests {
		if got := clampIncidentLimit(test.in); got != test.want {
			t.Errorf("clampIncidentLimit(%d) = %d, want %d", test.in, got, test.want)
		}
	}
}
