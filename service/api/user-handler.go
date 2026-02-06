package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubilaykoccc/Wasa/service/api/reqcontext"
)

// setMyUserName updates the user's username
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SetUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Name) < 3 || len(req.Name) > 16 {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	err := rt.db.SetMyUserName(ctx.UserID, req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to set username")
		// Handle conflict if username is already taken (UNIQUE constraint)
		http.Error(w, "Username already in use or invalid", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setMyPhoto updates the user's photo
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photoData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = rt.db.SetMyPhoto(ctx.UserID, photoData)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to set photo")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// searchUsers searches for users
func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	users, err := rt.db.SearchUsers(query)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to search users")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	resp := make([]UserResponse, len(users))
	for i, u := range users {
		resp[i] = FromDatabaseUser(u)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
