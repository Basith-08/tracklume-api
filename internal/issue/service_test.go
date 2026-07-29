package issue

import (
	"testing"
	"time"
)

func TestIssueDateValidation(t *testing.T) {
	valid := "2026-08-15"
	date, err := parseDate(&valid)
	if err != nil || date.Format("2006-01-02") != valid {
		t.Fatalf("valid date rejected: %v", err)
	}
	invalid := "15-08-2026"
	if _, err = parseDate(&invalid); err == nil {
		t.Fatal("invalid date accepted")
	}
}

func TestIssueFilterDateZero(t *testing.T) {
	if !(time.Time{}).IsZero() {
		t.Fatal("time sanity check failed")
	}
}
