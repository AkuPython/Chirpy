package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func handlerChirp(w http.ResponseWriter, r *http.Request) {
	type chirpParameters struct {
		Body string `json:"body"`
	}
	
	type cleanChirpParameters struct {
		Body string `json:"cleaned_body"`
	}
	
	type errorParameters struct {
		Body string `json:"error"`
	}
	
	type successParameters struct {
		Body bool `json:"valid"`
	}

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	chirp := chirpParameters{}
	err := decoder.Decode(&chirp)

	var resp any

	bad_words := make(map[string]string)
	bad_words["kerfuffle"] = "****"
	bad_words["sharbert"] = "****"
	bad_words["fornax"] = "****"


	if err != nil {
		w.WriteHeader(400)
		resp = errorParameters{Body: "Something went wrong"}
	} else if len(chirp.Body) > 140 {
		w.WriteHeader(400)
		resp = errorParameters{Body: "Chirp is too long"}
	} else {
		w.WriteHeader(200)
		// resp = successParameters{Body: true}
		var cleaned []string
		for _, word := range(strings.Fields(chirp.Body)) {
			if _, ok := bad_words[strings.ToLower(word)]; ok {
				cleaned = append(cleaned, "****")
			} else {
				cleaned = append(cleaned, word)
			}
		resp = cleanChirpParameters{Body: strings.Join(cleaned, " ")}

		}

	}
	dat, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	
	w.Write(dat)
}


func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")	
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}


func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, r *http.Request) {
	hits_html := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte(hits_html))
}

func (cfg *apiConfig) handlerResetMetrics(w http.ResponseWriter, r *http.Request) { 
	cfg.fileserverHits.Store(0)

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")	

	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("Hits reset! Hits: %d", cfg.fileserverHits.Load())))
}

