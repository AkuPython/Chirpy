package main

import (
	"log"
	"net/http"
)

func main()  {
	const port = "8080"
	const rootPath = "."
	
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app",http.FileServer(http.Dir(rootPath))))
	mux.HandleFunc("/healthz/", handlerReadiness)


	srv := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	

	log.Printf("Serving files from: '%s' on port: %s\n", rootPath, port)
	log.Fatal(srv.ListenAndServe())

}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")	
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

