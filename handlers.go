package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AkuPython/Chirpy/internal/database"
	"github.com/google/uuid"
)

type errorParameters struct {
	Body string `json:"error"`
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
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}

	cfg.fileserverHits.Store(0)

	cfg.db.DeleteUsers(r.Context())
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")	

	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("Hits reset! Hits: %d\nUsers Deleted!", cfg.fileserverHits.Load())))
}

func handlerChirp(w http.ResponseWriter, r *http.Request) {
	type chirpParameters struct {
		Body string `json:"body"`
	}
	
	type cleanChirpParameters struct {
		Body string `json:"cleaned_body"`
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

func (cfg *apiConfig) handlerAddUser(w http.ResponseWriter, r *http.Request) {
	type userCreateParameters struct {
		Body string `json:"email"`
	}
	type userParameters struct {
		Id uuid.UUID `json:"id"`
		Created time.Time `json:"created_at"`
		Updated time.Time `json:"updated_at"`
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	email := userCreateParameters{}
	err := decoder.Decode(&email)

	var resp any
	var user database.User
	
	if err == nil {
		email := sql.NullString{String: email.Body, Valid: true}
		user, err = cfg.db.CreateUser(r.Context(), email)
		if err != nil {
			fmt.Println("here")
		}
	}

	if err != nil {
		w.WriteHeader(400)
		resp = errorParameters{Body: fmt.Sprintf("Something went wrong: %s", err)}
	} else {
		w.WriteHeader(201)
		resp = userParameters{
			Id: user.ID,
			Created: user.CreatedAt.Time,
			Updated: user.UpdatedAt.Time,
			Email: user.Email.String,
		}
	}

	w.Header().Add("Content-Type", "application/json")	
	dat, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(dat)
}
