package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AkuPython/Chirpy/internal/auth"
	"github.com/AkuPython/Chirpy/internal/database"
	"github.com/google/uuid"
)


type userCreateParameters struct {
	Password string `json:"password"`
	Email string `json:"email"`
}

type errorParameters struct {
	Body string `json:"error"`
}

type chirpParameters struct {
	Body string `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}

type cleanChirpParameters struct {
	Body string `json:"cleaned_body"`
}

type successParameters struct {
	Body bool `json:"valid"`
}

type tokenParameters struct {
	Body string `json:"token"`
}


type userParameters struct {
	Id uuid.UUID `json:"id"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
	Email string `json:"email"`
	Token string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IsRed bool `json:"is_chirpy_red"`
}

type Chirp struct {
	Id uuid.UUID `json:"id"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}


func convertDbChirp (dbChirp database.Chirp) Chirp {
	jsonChirp := Chirp{
		Id: dbChirp.ID,
		Created: dbChirp.CreatedAt,
		Updated: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		UserId: dbChirp.UserID,
	}
	return jsonChirp
}

func convertDbUser (userDB database.User) userParameters {
	user := userParameters{
		Id: userDB.ID,
		Created: userDB.CreatedAt,
		Updated: userDB.UpdatedAt,
		Email: userDB.Email,
		IsRed: userDB.IsChirpyRed,
	}
	return user
}

func writeJSON(w http.ResponseWriter, c int, resp any) {
	dat, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	
	w.WriteHeader(c)
	w.Write(dat)
	return
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

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpId := r.PathValue("chirpId")
	chirpUUID, err := uuid.Parse(chirpId)
	if err != nil {
		writeJSON(w, 400, errorParameters{Body: fmt.Sprintf("Error Converting chirpId to UUID: %v\nErr: %v", chirpId, err)})
		return
	}
	chirp, err := cfg.db.ChirpGet(r.Context(), chirpUUID)
	if err != nil {
		writeJSON(w, 404, errorParameters{Body: fmt.Sprintf("Error Getting ChirpID: %v\nErr: %v", chirpId, err)})
		return
	}
	writeJSON(w, 200, convertDbChirp(chirp))
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.ChirpsGet(r.Context())
	if err != nil {
		writeJSON(w, 400, errorParameters{Body: fmt.Sprintf("Error Getting Chirps: %v", err)})
		return
	}
	jsonChirps := []Chirp{}
	for _, chirp := range chirps {
		jsonChirps = append(jsonChirps, convertDbChirp(chirp))
	}
	writeJSON(w, 200, jsonChirps)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userParams := userCreateParameters{}
	err := decoder.Decode(&userParams)

	if err != nil {
		writeJSON(w, 400, errorParameters{Body: "Invalid Login Request"})
		return
	}
	userDB, err := cfg.db.GetUserByEmail(r.Context(), userParams.Email)
	if err != nil {
		writeJSON(w, 400, errorParameters{Body: "Could not find user by email"})
		return
	}
	
	err = auth.CheckPasswordHash(userDB.HashedPassword, userParams.Password)

	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "Invalid Password"})
		return
	}
	expires := time.Duration(3600 * int(time.Second))
	token, err := auth.MakeJWT(userDB.ID, cfg.jwt_secret, expires)
	
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "Could not generate Token!"})
		return
	}

	refresh_token, err := auth.MakeRefreshToken()
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "Could not generate Refresh Token!"})
		return
	}
	refresh_token2, err := cfg.db.RefreshTokenAdd(r.Context(), database.RefreshTokenAddParams{
		Token: refresh_token,
		UserID: userDB.ID,
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour)})
	
	if err != nil || refresh_token != refresh_token2 {
		writeJSON(w, 401, errorParameters{Body: "Refresh Token DB Issue!"})
		cfg.db.RefreshTokenRevoke(r.Context(), refresh_token)
		return
	}

	returnParams := convertDbUser(userDB)
	returnParams.Token = token
	returnParams.RefreshToken = refresh_token

	writeJSON(w, 200, returnParams)

}
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rtoken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "No Auth Header in request"})
		return
	}
	refresh_token, err := cfg.db.RefreshTokenGet(r.Context(), rtoken)
	if err != nil || refresh_token.RevokedAt.Valid == true || time.Now().UTC().After(refresh_token.ExpiresAt) {
		if refresh_token.ExpiresAt.After(time.Now().UTC()) {
			cfg.db.RefreshTokenRevoke(r.Context(), refresh_token.Token)
		}
		writeJSON(w, 401, errorParameters{Body: "Refresh token invalid or expired!"})
		return
	}
	expires := time.Duration(3600 * int(time.Second))
	token, err := auth.MakeJWT(refresh_token.UserID, cfg.jwt_secret, expires)
	
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "Could not generate Token!"})
		return
	}

	writeJSON(w, 200, tokenParameters{Body: token})

}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "application/json")

	rtoken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "No Auth Header in request"})
		return
	}
	err = cfg.db.RefreshTokenRevoke(r.Context(), rtoken)
	if err != nil {
		writeJSON(w, 400, errorParameters{Body: "DB Error, could not revoke token"})
		return
	}
	w.WriteHeader(204)

}

func (cfg *apiConfig) handlerAddChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "No Auth Header in request"})
		return
	}
	
	token_user, err := auth.ValidateJWT(token, cfg.jwt_secret)
	// _, err = auth.ValidateJWT(token, cfg.jwt_secret)
	
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "Invalid or expired token"})
		return
	}


	decoder := json.NewDecoder(r.Body)
	newChirp := chirpParameters{}
	err = decoder.Decode(&newChirp)


	var resp any

	bad_words := make(map[string]string)
	bad_words["kerfuffle"] = "****"
	bad_words["sharbert"] = "****"
	bad_words["fornax"] = "****"


	if err != nil {
		resp = errorParameters{Body: "Something went wrong"}
		writeJSON(w, 400, resp)
		return
	}

	if len(newChirp.Body) > 140 {
		resp = errorParameters{Body: "Chirp is too long"}
		writeJSON(w, 400, resp)
		return
	}
	
	var cleaned []string
	for _, word := range(strings.Fields(newChirp.Body)) {
		if _, ok := bad_words[strings.ToLower(word)]; ok {
			cleaned = append(cleaned, "****")
		} else {
			cleaned = append(cleaned, word)
		}
	}
	var chirp database.ChirpAddParams
	chirp.Body = strings.Join(cleaned, " ")
	chirp.UserID = token_user

	added_chirp, err := cfg.db.ChirpAdd(r.Context(), chirp)
	if err != nil {
		resp = errorParameters{Body: "Adding Chirp Failed!"}
		writeJSON(w, 400, resp)
		return
	}
	jsonChirp := convertDbChirp(added_chirp)
	
	writeJSON(w, 201, jsonChirp)

}

func (cfg *apiConfig) handlerDeleteChirps(w http.ResponseWriter, r *http.Request) {

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "No Auth Header in request"})
		return
	}
	
	token_user, err := auth.ValidateJWT(token, cfg.jwt_secret)
	// _, err = auth.ValidateJWT(token, cfg.jwt_secret)
	
	if err != nil {
		writeJSON(w, 403, errorParameters{Body: "Invalid or expired token"})
		return
	}

	chirpId := r.PathValue("chirpId")
	chirpUUID, err := uuid.Parse(chirpId)
	if err != nil {
		writeJSON(w, 400, errorParameters{Body: fmt.Sprintf("Error Converting chirpId to UUID: %v\nErr: %v", chirpId, err)})
		return
	}
	chirp, err := cfg.db.ChirpGet(r.Context(), chirpUUID)
	if err != nil {
		writeJSON(w, 404, errorParameters{Body: fmt.Sprintf("Error Getting ChirpID: %v\nErr: %v", chirpId, err)})
		return
	}
	if chirp.UserID != token_user {
		writeJSON(w, 403, errorParameters{Body: "Wrong user for delete!"})
		return

	}
	cfg.db.ChirpDelete(r.Context(), chirpUUID)
	w.WriteHeader(204)
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")	
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerAddUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")	

	decoder := json.NewDecoder(r.Body)
	user := userCreateParameters{}
	err := decoder.Decode(&user)

	var resp any
	var newUser database.CreateUserRow
	
	if err == nil {
		email := user.Email
		password, err := auth.HashPassword(user.Password)
		if err != nil {
			writeJSON(w, 500, errorParameters{Body: "PW Hash fail!"})
			return
		}
		userParam := database.CreateUserParams{Email: email, HashedPassword: password}
		newUser, err = cfg.db.CreateUser(r.Context(), userParam)
	}

	if err != nil {
		w.WriteHeader(400)
		resp = errorParameters{Body: fmt.Sprintf("Something went wrong: %s", err)}
	} else {
		w.WriteHeader(201)
		resp = userParameters{
			Id: newUser.ID,
			Created: newUser.CreatedAt,
			Updated: newUser.UpdatedAt,
			Email: newUser.Email,
			IsRed: newUser.IsChirpyRed,
		}
	}

	dat, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(dat)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")	

	decoder := json.NewDecoder(r.Body)
	user := userCreateParameters{}
	err := decoder.Decode(&user)
	if err != nil {
		writeJSON(w, 400, errorParameters{Body: "Invalid Body"})
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "No Auth Header in request"})
		return
	}
	token_user, err := auth.ValidateJWT(token, cfg.jwt_secret)
	
	if err != nil {
		writeJSON(w, 401, errorParameters{Body: "Invalid or expired token"})
		return
	}

	var resp any
	
	var updatedUser database.User

	password, err := auth.HashPassword(user.Password)
	if err != nil {
		writeJSON(w, 500, errorParameters{Body: "PW Hash fail!"})
		return
	}
	updateParams := database.UpdateOneUserParams{ID: token_user, Email: user.Email, HashedPassword: password}
	updatedUser, err = cfg.db.UpdateOneUser(r.Context(), updateParams)
	if err != nil {
		writeJSON(w, 500, errorParameters{Body: "DB Update failed!"})
		return
	}
	
	w.WriteHeader(200)
	resp = convertDbUser(updatedUser)

	dat, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(dat)
}
