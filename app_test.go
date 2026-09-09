package main

import (
	"testing"
)

func TestAppEncryptDecrypt(t *testing.T) {
	app := NewApp()
	plaintext := "Hello, Wails v3!"
	password := "SecretPassword123"
	iterations := 10000

	encrypted, err := app.Encrypt(plaintext, password, iterations)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := app.Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected decrypted text %q, got %q", plaintext, decrypted)
	}

	// Test invalid password
	_, err = app.Decrypt(encrypted, "WrongPassword")
	if err == nil {
		t.Error("Expected error for wrong password, got nil")
	}

	// Test iteration bounds
	_, err = app.Encrypt(plaintext, password, minIterations-1)
	if err == nil {
		t.Error("Expected error for iterations < minIterations, got nil")
	}

	_, err = app.Encrypt(plaintext, password, maxIterations+1)
	if err == nil {
		t.Error("Expected error for iterations > maxIterations, got nil")
	}
}
