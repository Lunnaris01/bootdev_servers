package auth

import (
    "testing"
    "time"
    "github.com/google/uuid"
)


func TestValidateJWT(t *testing.T) {
    // Create a test UUID and secret
    userID := uuid.New()
    secret := "your-test-secret"
    
    // Test valid token
    t.Run("valid token", func(t *testing.T) {
        // Create token that expires in 1 hour
        token, err := MakeJWT(userID, secret, time.Hour)
        if err != nil {
            t.Fatal("failed to create token:", err)
        }

        // Validate the token
        gotUserID, err := ValidateJWT(token, secret)
        if err != nil {
            t.Fatal("failed to validate token:", err)
        }
		if gotUserID != userID {
			t.Errorf("Expected ID %v, got %v", userID, gotUserID)
		}

    })

    // Test expired token
    t.Run("expired token", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, -time.Hour)
		if err != nil {
            t.Fatal("failed to create token:", err)
        }

        // Validate the token
        _, err = ValidateJWT(token, secret)
        if err == nil {
            t.Errorf("Expected Token to me Invalid but was valid")
        }
    })

    // Test wrong secret
    t.Run("wrong secret", func(t *testing.T) {

        token, err := MakeJWT(userID, secret, time.Hour)
        if err != nil {
            t.Fatal("failed to create token:", err)
        }

        // Validate the token
        _, err = ValidateJWT(token, "wrong-secret!")
        if err == nil {
            t.Errorf("Wrong Secret was accepted")
        }
    })
}