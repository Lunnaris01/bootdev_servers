package main

import (
	"net/http"
)

func main(){
	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr: "localhost:8080",
	}

	serveMux.Handle("/app/",http.StripPrefix("/app/",http.FileServer(http.Dir("."))))
	serveMux.HandleFunc("/healthz",healthHandler)
	server.ListenAndServe()


}


func healthHandler(w http.ResponseWriter, req *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}