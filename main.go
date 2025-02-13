package main

import (
	"net/http"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"io"
	"strings"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/Lunnaris01/bootdev_servers/internal/database"
	"github.com/Lunnaris01/bootdev_servers/internal/auth"
	"os"
	"database/sql"
	"log"
	"time"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	secretKey string

}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w,r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, req *http.Request){
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	content_html := 
	`<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>`
	content_to_serve := fmt.Sprintf(content_html,cfg.fileserverHits.Load())
	w.Write([]byte(content_to_serve))

}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, req *http.Request){
	if cfg.platform != "dev" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		w.Write([]byte("Reset is only possible in development mode"))	
		return 
	}
	cfg.fileserverHits.Store(0)
	cfg.dbQueries.DeleteAllUsers(req.Context())
	cfg.dbQueries.DeleteAllChirps(req.Context())
	cfg.dbQueries.DeleteAllRefreshTokens(req.Context())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("Reset successfull"))
}


func (cfg *apiConfig) postChirpsHandler (w http.ResponseWriter, req *http.Request){

	type req_body struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type res_body struct{
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	bearerToken, err := auth.GetBearerToken(req.Header)
	
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.secretKey)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	r_body := req_body{}
	r_data, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	err = json.Unmarshal(r_data,&r_body)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	r_body.Body = clean_chirp(r_body.Body)
	if len(r_body.Body)>140{
		w.WriteHeader(401)
		w.Write([]byte("Chirp too Long!"))
		return
	}
	dbChirp, err := cfg.dbQueries.CreateChirp(req.Context(),database.CreateChirpParams{r_body.Body,userID})
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	retChirp := res_body{
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		UserID: userID,
	}

	response_json, _ := json.Marshal(retChirp)
	w.WriteHeader(201)
	w.Write(response_json)



}

func clean_chirp(input_message string) string {
	badwords := []string{"kerfuffle","sharbert","fornax"}
	for _,word := range badwords{
		input_message = strings.ReplaceAll(input_message,word,"****")
		input_message = strings.ReplaceAll(input_message,strings.Title(word),"****")
	}
	return input_message
}



func (cfg *apiConfig) getChirpsHandler (w http.ResponseWriter, req *http.Request){
	chirps, err := cfg.dbQueries.GetAllChirps(req.Context())
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	type resChirp struct{
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	var res_chirps []resChirp

	for _, chirp := range chirps{
		res_chirps = append(res_chirps,resChirp{
			ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body: chirp.Body,
			UserID: chirp.UserID,
		})

	}
	
	response_json, err := json.Marshal(res_chirps)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(200)
	w.Write(response_json)
}

func (cfg *apiConfig) getChirpHandler (w http.ResponseWriter, req *http.Request){
	chirpIDStr := req.PathValue("chirpID")
	chirpIDUUID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}
	db_chirp, err := cfg.dbQueries.GetChirp(req.Context(),chirpIDUUID)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	type resChirp struct{
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	res_chirp := resChirp{
		ID: db_chirp.ID,
		CreatedAt: db_chirp.CreatedAt,
		UpdatedAt: db_chirp.UpdatedAt,
		Body: db_chirp.Body,
		UserID: db_chirp.UserID,
	}
	response_json, err := json.Marshal(res_chirp)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(200)
	w.Write(response_json)
}

func (cfg *apiConfig) deleteChirpHandler (w http.ResponseWriter, req *http.Request){
	//Check first if the Chirp exists!
	chirpIDStr := req.PathValue("chirpID")
	chirpIDUUID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}
	db_chirp, err := cfg.dbQueries.GetChirp(req.Context(),chirpIDUUID)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	bearerToken, err := auth.GetBearerToken(req.Header)
	
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.secretKey)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	if db_chirp.UserID != userID{
		w.WriteHeader(403)
		return
	}
	
	err = cfg.dbQueries.DeleteChirp(req.Context(),chirpIDUUID)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(204)

}


func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, req *http.Request){
	type addUserBody struct{
		Email string `json:"email"`
		Password string `json:"password"`
	}
	type User struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string	`json:"email"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}

	r_body := addUserBody{}
	r_data, err := io.ReadAll(req.Body)

	defer req.Body.Close()
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	err = json.Unmarshal(r_data,&r_body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	hashedPassword, err := auth.HashPassword(r_body.Password)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	db_user, err := cfg.dbQueries.CreateUser(
		req.Context(),
		database.CreateUserParams{
			Email: r_body.Email,
			HashedPassword: hashedPassword,
		})
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	ret_user := User {
		ID: db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email: db_user.Email,
		IsChirpyRed: db_user.IsChirpyRed,
	}
	response_json, _ := json.Marshal(ret_user)
	w.WriteHeader(201)
	w.Write(response_json)
}

func (cfg *apiConfig) loginUserHandler (w http.ResponseWriter, req *http.Request){
	type loginUserBody struct{
		Email string `json:"email"`
		Password string `json:"password"`
	}

	type User struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string	`json:"email"`
		Token 	  string	`json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsChirpyRed bool `json:"is_chirpy_red"`
	}


	r_body := loginUserBody{}
	r_data, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	err = json.Unmarshal(r_data,&r_body)
	// Ensure valid value was used or set default value of 1 hour for Duration
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	db_user, err := cfg.dbQueries.GetUserByMail(req.Context(),r_body.Email)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}
	err = auth.CheckPasswordHash(r_body.Password,db_user.HashedPassword)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}

	jwtToken, err := auth.MakeJWT(db_user.ID,cfg.secretKey,time.Duration(1)*time.Hour)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Failed to create JWT Token"))
		return
	}

	str_refreshToken, err := auth.MakeRefreshToken()

	refreshTokenParams := database.CreateRefreshTokenParams{
		Token:		str_refreshToken,
		ExpiresAt:	time.Now().Add(time.Duration(60*24)*time.Hour),
		UserID:		db_user.ID,
	}

	refreshToken, err := cfg.dbQueries.CreateRefreshToken(req.Context(),refreshTokenParams)
	if err != nil {
		fmt.Println("Error creating refresh token: %v", err)
		return 
	}



	ret_user := User {
		ID: db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email: db_user.Email,
		Token: jwtToken,
		RefreshToken: refreshToken.Token,
		IsChirpyRed: db_user.IsChirpyRed,
	}
	
	response_json, _ := json.Marshal(ret_user)
	w.WriteHeader(200)
	w.Write(response_json)



}

func (cfg *apiConfig) refreshAccessToken (w http.ResponseWriter, req *http.Request){
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Failed to retreive Refresh Token"))
		return
	}
	tokenUserID, err := cfg.dbQueries.GetUserFromRefreshToken(req.Context(),bearerToken)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid token"))
		return
	}

	jwtToken, err := auth.MakeJWT(tokenUserID,cfg.secretKey,time.Duration(1)*time.Hour)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Failed to create access token"))
		return
	}
	type ResponseBody struct {
		Token string `json:"token"`
	}

	res_body := ResponseBody{
		Token:  jwtToken,
	}
	response_json, _ := json.Marshal(res_body)
	w.WriteHeader(200)
	w.Write(response_json)
}

func (cfg *apiConfig) revokeRefreshToken (w http.ResponseWriter, req *http.Request){
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Failed to read Token"))
		return
	}

	err = cfg.dbQueries.RevokeTokenAccess(req.Context(),database.RevokeTokenAccessParams{bearerToken,time.Now()})
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Failed to revoke Access for Token"))
		return
	}
	w.WriteHeader(204)

}


func (cfg *apiConfig) updateUserHandler (w http.ResponseWriter, req *http.Request){
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Failed to retreive Refresh Token"))
		return
	}
	tokenUserID, err := auth.ValidateJWT(bearerToken,cfg.secretKey)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid token"))
		return
	}
	type updateUserBody struct{
		Password string `json:"password"`
		Email string `json:"email"`
	}

	r_body := updateUserBody{}
	r_data, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	err = json.Unmarshal(r_data,&r_body)

	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	new_hashedPassword, err := auth.HashPassword(r_body.Password)

	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}


	updated_user, err := cfg.dbQueries.UpdateUserPassAndMailByID(req.Context(),database.UpdateUserPassAndMailByIDParams{
		ID: tokenUserID,
		Email: r_body.Email,
		HashedPassword: new_hashedPassword,
	})

	if err != nil{
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	type User struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string	`json:"email"`
	}

	ret_user := User {
		ID: updated_user.ID,
		CreatedAt: updated_user.CreatedAt,
		UpdatedAt: updated_user.UpdatedAt,
		Email: updated_user.Email,
	}
	response_json, _ := json.Marshal(ret_user)
	w.WriteHeader(200)
	w.Write(response_json)

}

func (cfg *apiConfig) subscribeUser (w http.ResponseWriter, req *http.Request){
	type subscribeUserBody struct{
		Event string `json:"event"`
		Data struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	r_body := subscribeUserBody{}
	r_data, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}

	err = json.Unmarshal(r_data,&r_body)

	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	if r_body.Event != "user.upgraded"{
		w.WriteHeader(204)
		return
	}
	userUUID, err := uuid.Parse(r_body.Data.UserID)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	err = cfg.dbQueries.SubscribeUser(req.Context(),userUUID)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(204)




}
func main(){
	godotenv.Load()

	env_dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", env_dbURL)
	if err != nil {
		log.Fatalf("Failed to load database")
	}
	dbQueries := database.New(db)
	env_platform := os.Getenv("PLATFORM")
	env_secretKey := os.Getenv("SECRET_KEY")

	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr: "localhost:8080",
	}
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries: dbQueries,
		platform: env_platform,
		secretKey: env_secretKey,
	}

	serveMux.Handle("/app/",http.StripPrefix("/app/",apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz",healthHandler)
	serveMux.HandleFunc("GET /admin/metrics",apiCfg.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset",apiCfg.resetHandler)
	serveMux.HandleFunc("POST /api/chirps",apiCfg.postChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpHandler)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}",apiCfg.deleteChirpHandler)
	serveMux.HandleFunc("POST /api/users",apiCfg.addUserHandler)
	serveMux.HandleFunc("PUT /api/users", apiCfg.updateUserHandler)
	serveMux.HandleFunc("POST /api/login",apiCfg.loginUserHandler)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.refreshAccessToken)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.revokeRefreshToken)
	serveMux.HandleFunc("POST /api/polka/webhooks", apiCfg.subscribeUser)

	server.ListenAndServe()


}


func healthHandler(w http.ResponseWriter, req *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

