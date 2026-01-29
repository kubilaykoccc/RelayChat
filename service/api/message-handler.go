package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
	"github.com/julienschmidt/httprouter"
)

// sendMessage sends a message
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("conversationId")
	// Try parsing as integer
	var conversationId uint64
	var err error

	possibleId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	// Smart logic: Check if conversation exists
	_, err = rt.db.GetConversation(possibleId, ctx.UserID)
	if err != nil {
		// Not found or not allowed.
		// Try to interpret as User ID for 1-on-1
		// Check if possibleId matches a USER?
		// But in numeric system, UserID and ConversationID are just integers.
		// We can try to CreateConversation(ctx.UserID, possibleId).
		// If possibleId refers to a User, this finds/creates the conversation.
		// If it's just a random number, it fails (foreign key error on user_id).

		convo, createErr := rt.db.CreateConversation(ctx.UserID, possibleId)
		if createErr == nil {
			conversationId = convo.ID
		} else {
			// Failed to create conversation implies not a valid user ID or DB error
			ctx.Logger.WithError(err).Error("conversation not found and failed to create 1-on-1")
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
	} else {
		conversationId = possibleId
	}

	var req SendMessageRequest
	var photoData []byte

	// Check if Multipart
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(10 << 20) // 10 MB limit
		if err != nil {
			http.Error(w, "Invalid Multipart Form", http.StatusBadRequest)
			return
		}

		req.Content = r.FormValue("content") // Text content
		req.Type = r.FormValue("type")
		if req.Type == "" {
			req.Type = "image" // Default to image if multipart? Or mixed?
		}

		replyIdStr := r.FormValue("replyToId")
		if replyIdStr != "" {
			rid, _ := strconv.ParseUint(replyIdStr, 10, 64)
			req.ReplyToID = rid
		}

		// Read file
		file, _, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			photoData, err = io.ReadAll(file)
			if err != nil {
				http.Error(w, "Error reading file", http.StatusInternalServerError)
				return
			}
		}
	} else {
		// JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	msg := database.Message{
		ConversationID: conversationId,
		SenderID:       ctx.UserID,
		Type:           req.Type,
		Content:        req.Content,
		Photo:          photoData,
	}
	if req.ReplyToID != 0 {
		msg.ReplyToID = &req.ReplyToID
	}

	sentMsg, err := rt.db.SendMessage(msg)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to send message")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := FromDatabaseMessage(sentMsg)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// forwardMessage forwards a message
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
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

	var req ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg, err := rt.db.ForwardMessage(conversationId, req.ForwardedMessageID, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to forward message")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := FromDatabaseMessage(msg)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// deleteMessage deletes a message
func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("messageId")
	messageId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Message ID", http.StatusBadRequest)
		return
	}

	err = rt.db.DeleteMessage(messageId, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to delete message")
		http.Error(w, "Not found or forbidden", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// commentMessage reacts to a message
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("messageId")
	messageId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Message ID", http.StatusBadRequest)
		return
	}

	var req CommentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reaction := database.Reaction{
		MessageID: messageId,
		UserID:    ctx.UserID,
		Emoji:     req.Emoji,
	}

	res, err := rt.db.CommentMessage(reaction)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to comment")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := FromDatabaseReaction(res)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// uncommentMessage removes a reaction
func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := ps.ByName("reactionId")
	reactionId, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Reaction ID", http.StatusBadRequest)
		return
	}

	err = rt.db.UncommentMessage(reactionId, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to uncomment")
		http.Error(w, "Not found or forbidden", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleMessageAction dispatches between forwardMessage and commentMessage
func (rt *_router) handleMessageAction(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	action := ps.ByName("action") // starts with /

	if action == "/forwarded" {
		rt.forwardMessage(w, r, ps, ctx)
		return
	}

	// Check for .../:messageId/reactions
	// Format: /<messageId>/reactions
	parts := strings.Split(strings.TrimPrefix(action, "/"), "/")
	if len(parts) == 2 && parts[1] == "reactions" {
		// Create new params with messageId
		messageId := parts[0]
		newParams := make(httprouter.Params, len(ps)+1)
		copy(newParams, ps)
		newParams[len(ps)] = httprouter.Param{Key: "messageId", Value: messageId}

		rt.commentMessage(w, r, newParams, ctx)
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}
