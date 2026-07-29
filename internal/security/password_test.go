package security

import "testing"

func TestPasswordHashAndVerification(t *testing.T) {
	hash, err := HashPassword("Password123!")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "Password123!" || !VerifyPassword(hash, "Password123!") {
		t.Fatal("password hash did not verify")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("wrong password verified")
	}
}
