package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/kubilaykoccc/Wasa/service/api/reqcontext"
	"github.com/kubilaykoccc/Wasa/service/database"
)

// getMyConversations returns the list of conversations
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := rt.db.GetMyConversations(ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to get conversations")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]ConversationUnreadResponse, len(conversations))
	for i, c := range conversations {
		resp[i] = FromDatabaseConversationUnread(c)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// getConversation returns conversation details
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("conversationId")
	conversationId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	details, err := rt.db.GetConversation(conversationId, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to get conversation details")
		http.Error(w, "Not checking error type precisely but likely 404 or 403", http.StatusNotFound)
		return
	}

	// Convert to response
	members := make([]UserResponse, len(details.Members))
	for i, m := range details.Members {
		members[i] = FromDatabaseUser(m)
	}
	messages := make([]MessageResponse, len(details.Messages))
	for i, m := range details.Messages {
		messages[i] = FromDatabaseMessage(m)
	}

	// Construct response

	resp := struct {
		ID       uint64            `json:"id"`
		Name     string            `json:"name,omitempty"`
		IsGroup  bool              `json:"isGroup"`
		Photo    []byte            `json:"photo,omitempty"`
		OwnerID  uint64            `json:"ownerId"`
		Members  []UserResponse    `json:"members"`
		Messages []MessageResponse `json:"messages"`
	}{
		ID:       details.ID,
		Name:     details.Name,
		IsGroup:  details.IsGroup,
		Photo:    details.Photo,
		OwnerID:  details.OwnerID,
		Members:  members,
		Messages: messages,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// setGroupName sets the group name
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("conversationId")
	conversationId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	var req SetGroupNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Name) < 1 {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}

	err = rt.db.SetGroupName(conversationId, req.Name)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to set group name")
		http.Error(w, "Forbidden or Not Found", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// setGroupPhoto sets the group photo
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("conversationId")
	conversationId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	photoData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = rt.db.SetGroupPhoto(conversationId, photoData)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to set group photo")
		http.Error(w, "Forbidden or Not Found", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// addToGroup adds a user
func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("conversationId")
	conversationId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	userIdStr := ps.ByName("userId")
	userId, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	err = rt.db.AddToGroup(conversationId, userId)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to add user to group")
		http.Error(w, "Forbidden or Not Found", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// leaveGroup removes me
func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("conversationId")
	conversationId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	err = rt.db.LeaveGroup(conversationId, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to leave group")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// createConversation creates a new conversation (group)
func (rt *_router) createConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var conversation database.Conversation
	var err error

	if req.Name != "" {
		// Group
		conversation, err = rt.db.CreateGroup(req.Name, ctx.UserID)
	} else {
		// 1-on-1
		// Check members is int array now
		if len(req.Members) == 1 {
			conversation, err = rt.db.CreateConversation(ctx.UserID, req.Members[0])
		} else {
			http.Error(w, "Missing group name or invalid members for 1-on-1", http.StatusBadRequest)
			return
		}
	}

	if err != nil {
		ctx.Logger.WithError(err).Error("failed to create conversation")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add other members if group
	if req.Name != "" && len(req.Members) > 0 {
		for _, memberId := range req.Members {
			_ = rt.db.AddToGroup(conversation.ID, memberId)
		}
	}

	// Fetch details to return
	details, err := rt.db.GetConversation(conversation.ID, ctx.UserID)
	if err != nil {
		http.Error(w, "Error fetching created conversation", http.StatusInternalServerError)
		return
	}

	members := make([]UserResponse, len(details.Members))
	for i, m := range details.Members {
		members[i] = FromDatabaseUser(m)
	}
	messages := make([]MessageResponse, len(details.Messages))
	for i, m := range details.Messages {
		messages[i] = FromDatabaseMessage(m)
	}

	resp := struct {
		ID       uint64            `json:"id"`
		Name     string            `json:"name,omitempty"`
		IsGroup  bool              `json:"isGroup"`
		Photo    []byte            `json:"photo,omitempty"`
		OwnerID  uint64            `json:"ownerId"`
		Members  []UserResponse    `json:"members"`
		Messages []MessageResponse `json:"messages"`
	}{
		ID:       details.ID,
		Name:     details.Name,
		IsGroup:  details.IsGroup,
		Photo:    details.Photo,
		OwnerID:  details.OwnerID,
		Members:  members,
		Messages: messages,
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
