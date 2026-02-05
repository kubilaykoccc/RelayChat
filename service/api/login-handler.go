package api

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kubilaykoccc/Wasa/service/api/reqcontext"
)

// doLogin logs in the user (or registers them)
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate (simple validation)
	if len(req.Name) < 3 || len(req.Name) > 16 {
		http.Error(w, "Invalid username length", http.StatusBadRequest)
		return
	}

	id, err := rt.db.DoLogin(req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("login failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		Identifier: id,
	}
	w.WriteHeader(http.StatusCreated) // 201 Created
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
