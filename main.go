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

	r_body := req_body{}
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
	r_body.Body = clean_chirp(r_body.Body)
	if len(r_body.Body)>140{
		w.WriteHeader(400)
		w.Write([]byte("Chirp too Long!"))
		return
	}
	dbChirp, err := cfg.dbQueries.CreateChirp(req.Context(),database.CreateChirpParams{r_body.Body,r_body.UserID})
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	retChirp := res_body{
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		UserID: dbChirp.UserID,
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


	ret_user := User {
		ID: db_user.ID,
		CreatedAt: db_user.CreatedAt,
		UpdatedAt: db_user.UpdatedAt,
		Email: db_user.Email,
	}
	
	response_json, _ := json.Marshal(ret_user)
	w.WriteHeader(200)
	w.Write(response_json)



}



func main(){
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to load database")
	}
	dbQueries := database.New(db)
	env_platform := os.Getenv("PLATFORM")


	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr: "localhost:8080",
	}
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries: dbQueries,
		platform: env_platform,
	}

	serveMux.Handle("/app/",http.StripPrefix("/app/",apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz",healthHandler)
	serveMux.HandleFunc("GET /admin/metrics",apiCfg.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset",apiCfg.resetHandler)
	serveMux.HandleFunc("POST /api/chirps",apiCfg.postChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpHandler)
	serveMux.HandleFunc("POST /api/users",apiCfg.addUserHandler)
	serveMux.HandleFunc("POST /api/login",apiCfg.loginUserHandler)
	server.ListenAndServe()


}


func healthHandler(w http.ResponseWriter, req *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

