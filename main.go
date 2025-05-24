package main

import (
	// "fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}


func main()  {
	const port = "8080"
	const rootPath = "."
	
	apiCfg := apiConfig{}

	mux := http.NewServeMux()
	fsHandler := http.StripPrefix("/app",http.FileServer(http.Dir(rootPath)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/validate_chirp", handlerChirp)
	
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetMetrics)


	srv := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	

	log.Printf("Serving files from: '%s' on port: %s\n", rootPath, port)
	log.Fatal(srv.ListenAndServe())

}

