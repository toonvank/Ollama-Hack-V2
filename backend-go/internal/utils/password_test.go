package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if hash == "" {
		t.Fatalf("Expected hash to not be empty")
	}
	
	if hash == password {
		t.Fatalf("Expected hash to be different from password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, _ := HashPassword(password)
	
	if !CheckPassword(password, hash) {
		t.Fatalf("Expected CheckPassword to return true for matching password")
	}
	
	if CheckPassword("wrongpassword", hash) {
		t.Fatalf("Expected CheckPassword to return false for incorrect password")
	}
}
