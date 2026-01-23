package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// setMyUserName updates the user's username
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
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
		// Could be 409 Conflict if name taken?
		// My DB implementation doesn't check unique constraint explicitly but existing user handling.
		// Wait, the table definition has `username TEXT NOT NULL UNIQUE`.
		// So `SetMyUserName` will fail if not unique.
		http.Error(w, "Username already in use or invalid", http.StatusConflict) // simplified
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setMyPhoto updates the user's photo
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
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
	if ctx.UserID == 0 {
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
