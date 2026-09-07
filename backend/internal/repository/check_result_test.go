package repository

import "testing"

func TestClampCheckResultLimit(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{in: 0, want: defaultCheckResultLimit},
		{in: -5, want: defaultCheckResultLimit},
		{in: 10, want: 10},
		{in: maxCheckResultLimit, want: maxCheckResultLimit},
		{in: maxCheckResultLimit + 1, want: maxCheckResultLimit},
	}

	for _, test := range tests {
		if got := clampCheckResultLimit(test.in); got != test.want {
			t.Errorf("clampCheckResultLimit(%d) = %d, want %d", test.in, got, test.want)
		}
	}
}
