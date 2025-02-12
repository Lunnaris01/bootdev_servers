package main

import (
	"net/http"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"io"
	"strings"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32

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
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("Reset successfull"))

}

func (cfg *apiConfig) validateChirpHandler (w http.ResponseWriter, req *http.Request){
	type http_body struct{
		Body string `json:"body"`
		Error string `json:"error"`
		Valid bool `json:"valid"`
		CleanedBody string `json:"cleaned_body"`
	}

	r_body := http_body{}
	r_data, err := io.ReadAll(req.Body)
	if err != nil {
		response_body := http_body{
			Error: "Something went wrong",
		}	
		response_json, _ := json.Marshal(response_body)
		w.WriteHeader(400)
		w.Write([]byte(response_json))
		return
	}
	defer req.Body.Close()
	err = json.Unmarshal(r_data,&r_body)
	if err != nil {
		response_body := http_body{
			Error: "Something went wrong",
		}	
		response_json, _ := json.Marshal(response_body)
		w.WriteHeader(400)
		w.Write(response_json)
		return
	}
	r_body.CleanedBody = clean_chirp(r_body.Body)
	if len(r_body.CleanedBody)>140{
		response_body := http_body{
			Error: "Chirp is too long",
		}	
		response_json, _ := json.Marshal(response_body)
		w.WriteHeader(400)
		w.Write(response_json)
		return
	}
	response_body := http_body{
		CleanedBody: r_body.CleanedBody,
		Valid: true,
	}	
	response_json, _ := json.Marshal(response_body)
	w.WriteHeader(200)
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


func main(){
	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr: "localhost:8080",
	}
	apiCfg := apiConfig{}

	serveMux.Handle("/app/",http.StripPrefix("/app/",apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	serveMux.HandleFunc("GET /api/healthz",healthHandler)
	serveMux.HandleFunc("GET /admin/metrics",apiCfg.metricsHandler)
	serveMux.HandleFunc("POST /admin/reset",apiCfg.resetHandler)
	serveMux.HandleFunc("POST /api/validate_chirp",apiCfg.validateChirpHandler)
	server.ListenAndServe()


}


func healthHandler(w http.ResponseWriter, req *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

