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

func TestHashPasswordUnique(t *testing.T) {
	password := "samepassword"
	
	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	// Each hash should be unique due to bcrypt's salt
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (bcrypt salt)")
	}
	
	// But both should verify correctly
	if !CheckPassword(password, hash1) {
		t.Error("Expected hash1 to verify correctly")
	}
	if !CheckPassword(password, hash2) {
		t.Error("Expected hash2 to verify correctly")
	}
}

func TestCheckPasswordEmptyHash(t *testing.T) {
	result := CheckPassword("anypassword", "")
	if result {
		t.Error("Expected CheckPassword to return false for empty hash")
	}
}

func TestCheckPasswordEmptyPassword(t *testing.T) {
	hash, err := HashPassword("validpassword")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	result := CheckPassword("", hash)
	if result {
		t.Error("Expected CheckPassword to return false for empty password")
	}
}

func TestCheckPasswordInvalidHash(t *testing.T) {
	result := CheckPassword("anypassword", "invalidhash")
	if result {
		t.Error("Expected CheckPassword to return false for invalid hash")
	}
}

func TestHashPasswordSpecialCharacters(t *testing.T) {
	passwords := []string{
		"p@$$w0rd!",
		"你好世界",
		"emoji😀password",
		"spaces in password",
		"tab\there",
		"newline\npassword",
	}

	for _, password := range passwords {
		hash, err := HashPassword(password)
		if err != nil {
			t.Errorf("Failed to hash password '%s': %v", password, err)
			continue
		}
		
		if !CheckPassword(password, hash) {
			t.Errorf("Expected CheckPassword to return true for '%s'", password)
		}
	}
}

func TestHashPasswordMinimumLength(t *testing.T) {
	// Test with single character
	hash, err := HashPassword("a")
	if err != nil {
		t.Fatalf("Failed to hash single character: %v", err)
	}
	
	if !CheckPassword("a", hash) {
		t.Error("Expected CheckPassword to return true for single character")
	}
}

func TestCheckPasswordCaseSensitive(t *testing.T) {
	password := "CaseSensitive"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	// Same password should work
	if !CheckPassword("CaseSensitive", hash) {
		t.Error("Expected exact password to match")
	}
	
	// Different case should fail
	if CheckPassword("casesensitive", hash) {
		t.Error("Expected different case to not match")
	}
	if CheckPassword("CASESENSITIVE", hash) {
		t.Error("Expected different case to not match")
	}
}

func TestHashPasswordBenchmark(t *testing.T) {
	// Just ensure hashing multiple passwords works
	for i := 0; i < 5; i++ {
		password := "password" + string(rune('0'+i))
		hash, err := HashPassword(password)
		if err != nil {
			t.Errorf("Failed to hash password %d: %v", i, err)
		}
		if hash == "" {
			t.Errorf("Empty hash for password %d", i)
		}
	}
}
