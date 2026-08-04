package project

import "testing"

func TestRoleAuthorization(t *testing.T) {
	tests := []struct {
		role, action string
		allowed      bool
	}{
		{"owner", "archive", true}, {"admin", "archive", false}, {"member", "issue_write", true}, {"viewer", "issue_write", false}, {"viewer", "read", true}, {"admin", "member_manage", true},
	}
	for _, test := range tests {
		if got := RoleAllows(test.role, test.action); got != test.allowed {
			t.Errorf("RoleAllows(%q,%q)=%v, want %v", test.role, test.action, got, test.allowed)
		}
	}
}
