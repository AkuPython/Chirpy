package main

import (
	// "fmt"
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/AkuPython/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
}


func main()  {
	godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("DB open failed! ", err)
	}
	dbQueries := database.New(db)
	
	const port = "8080"
	const rootPath = "."
	
	apiCfg := apiConfig{db: dbQueries, platform: platform}


	mux := http.NewServeMux()
	fsHandler := http.StripPrefix("/app",http.FileServer(http.Dir(rootPath)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/validate_chirp", handlerChirp)
	
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetMetrics)
	mux.HandleFunc("POST /api/users", apiCfg.handlerAddUser)


	srv := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	

	log.Printf("Serving files from: '%s' on port: %s\n", rootPath, port)
	log.Fatal(srv.ListenAndServe())

}

