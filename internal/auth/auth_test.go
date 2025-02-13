package auth

import (
    "testing"
    "time"
    "github.com/google/uuid"
	"net/http"
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


func TestBearerToken(t *testing.T) {
    // Create a test UUID and secret
    
    // Test valid token
    t.Run("Test trimming", func(t *testing.T) {
		//add authorization to header
		header := http.Header{}
        header.Add("Authorization", "Bearer  Thistokenisanewtoken  ")

        bearerToken, err := GetBearerToken(header)
        if err != nil {
            t.Fatal("Failed to get bearer Token:", err)
        }

        expectedToken := "Thistokenisanewtoken"
        if bearerToken != expectedToken {
            t.Errorf("Unexpected Token: got %v, want %v", bearerToken, expectedToken)
        }
    })

    t.Run("Test valid", func(t *testing.T) {
		//add authorization to header
		header := http.Header{}

        header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")

        bearerToken, err := GetBearerToken(header)
        if err != nil {
            t.Fatal("Failed to get bearer Token:", err)
        }

        expectedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
        if bearerToken != expectedToken {
            t.Errorf("Unexpected Token: got %v, want %v", bearerToken, expectedToken)
        }
    })

    t.Run("Test wrong key", func(t *testing.T) {
		//add authorization to header
		header := http.Header{}

        header.Add("Authorize", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")

        bearerToken, err := GetBearerToken(header)
        if err == nil {
            t.Errorf("Expected an error and empty Token but got Token: %v,Error: %v\n",bearerToken,err)
        }
    })

    t.Run("Test misstyped bearer", func(t *testing.T) {
		//add authorization to header
		header := http.Header{}

        header.Add("Authorization", "Berer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")

        bearerToken, err := GetBearerToken(header)
        if err == nil {
            t.Errorf("Expected an error and empty Token but got Token: %v,Error: %v\n",bearerToken,err)
        }
    })


}