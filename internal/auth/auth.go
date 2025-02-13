package auth

import(
    "golang.org/x/crypto/bcrypt"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"crypto/rand"
	"encoding/hex"
)

func HashPassword(password string) (string, error){
	hashed_password, err := bcrypt.GenerateFromPassword([]byte(password),10)
	if err != nil{
		return  "",fmt.Errorf("failed to encrypt password - aborting")
	}
	err = bcrypt.CompareHashAndPassword(hashed_password,[]byte(password))
	if err != nil{
		return  "",fmt.Errorf("Encountered unknown fault when encrypting the password!")
	}

	return string(hashed_password), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash),[]byte(password))
	if err != nil{
		return  fmt.Errorf("Incorrect password")
	}
	return nil
} 

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){
	new_token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer: "chirpy",
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			Subject: userID.String(),
		})
	ss, err := new_token.SignedString([]byte(tokenSecret))
	return ss, err

}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil{
		return  uuid.UUID{},err
	}
	userIDStr, err := token.Claims.GetSubject()
	if err != nil{
		return  uuid.UUID{},err
	}
	userIDUUID, err := uuid.Parse(userIDStr)
	if err != nil{
		return  uuid.UUID{},err
	}
	return userIDUUID,nil
}

func GetBearerToken(headers http.Header) (string,error){
	keyString := "Bearer"
	bearerToken, err := GetAuthKey(headers,keyString)
	if err != nil{
		return "", err
	}
	return bearerToken, nil
}


func MakeRefreshToken() (string,error){
	key := make([]byte,32)
	rand.Read(key)
	key_string := hex.EncodeToString(key)
	return key_string, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	keyString := "ApiKey"
	apiToken, err := GetAuthKey(headers,keyString)
	if err != nil{
		return "", err
	}
	return apiToken, nil

}

func GetAuthKey(headers http.Header, keyName string) (string, error) {
	authString := headers.Get("Authorization")
	if authString == "" {
		return "", fmt.Errorf("No Authorization key or empty value!")
	}
	authString = strings.Trim(authString," ")
	if !strings.HasPrefix(authString,keyName){
		return "", fmt.Errorf("No \"%s\" substring in Authorization value!",keyName)
	}
	authToken := strings.Replace(authString,keyName, "",1)
	authToken = strings.Trim(authToken, " ")
	return authToken, nil
}