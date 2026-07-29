package dashboard

import "testing"

func TestCalculateProgress(t *testing.T) {
	if got := CalculateProgress(2, 8); got != 25 {
		t.Fatalf("got %v, want 25", got)
	}
	if got := CalculateProgress(1, 0); got != 0 {
		t.Fatalf("zero denominator returned %v", got)
	}
}
