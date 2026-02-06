package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/kubilaykoccc/Wasa/service/api/reqcontext"
	"github.com/kubilaykoccc/Wasa/service/database"
)

// sendMessage sends a message
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationId := ps.ByName("conversationId")
	if conversationId == "" {
		http.Error(w, "Invalid Conversation ID", http.StatusBadRequest)
		return
	}

	// Smart logic: Check if conversation exists
	// But first, we assume it's a UUID string.
	// If the frontend sends a user ID here (for implicit 1-on-1 creation), it's also a UUID string.

	_, err := rt.db.GetConversation(conversationId, ctx.UserID)
	if err != nil {
		// Not found as a conversation ID.
		// Try to interpret as User ID to find/create a 1-on-1 conversation.
		// Since both are strings, we just try to create.

		possibleUserId := conversationId
		convo, createErr := rt.db.CreateConversation(ctx.UserID, possibleUserId)
		if createErr == nil {
			conversationId = convo.ID
		} else {
			// Failed to create conversation implies not a valid user ID or DB error
			ctx.Logger.WithError(err).Error("conversation not found and failed to create 1-on-1")
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
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

		req.Text = r.FormValue("content") // Text content
		req.Type = r.FormValue("type")
		if req.Type == "" {
			req.Type = "image" // Default to image if multipart? Or mixed?
		}

		replyIdStr := r.FormValue("replyToId")
		if replyIdStr != "" {
			req.ReplyToID = replyIdStr
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

	// Handle content field fallback (frontend might send 'content' or 'text')
	finalContent := req.Text
	if finalContent == "" {
		finalContent = req.Content
	}

	msg := database.Message{
		ConversationID: conversationId,
		SenderID:       ctx.UserID,
		Type:           req.Type,
		Content:        finalContent,
		Photo:          photoData,
	}
	if req.ReplyToID != "" {
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
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageId := ps.ByName("messageId")
	if messageId == "" {
		http.Error(w, "Invalid Message ID", http.StatusBadRequest)
		return
	}

	var req ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Wait, the logic used to be: route param was messageId (to forward).
	// But `api-handler.go` (Step 204) matches route: `/messages/:messageId/forward` -> `h.forwardMessage`
	// This means `messageId` is the message being forwarded.
	// But `ForwardMessageRequest` struct (Step 205) has `ForwardedMessageID`.
	// The implementation in `messages.go` (Step 288) `ForwardMessage(conversationId, forwardedMessageId, senderId)`.

	// So the Request Body `ForwardMessageRequest` likely contains the TARGET `ConversationID`.
	// Wait, checking `structures.go` Step 258:
	// type ForwardMessageRequest struct {
	// 	ForwardedMessageID string `json:"forwardedMessageId"`
	// 	ConversationID     string `json:"conversationId"`
	// }

	// If the route is `/messages/:messageId/forward`, the `messageId` is in path.
	// The body should contain the destination `conversationId`.
	// The `ForwardedMessageID` in struct might be redundant or for the other route style.

	// Let's assume the body has valid `ConversationID`.
	// And we take `messageId` from path.

	// Oh wait, `api.yaml` (Step 248 plan) said: POST /messages/:messageId/forward
	// So `api-handler.go` is correct.
	// The struct `ForwardMessageRequest` has `ForwardedMessageID`... maybe from old code?
	// Let's use `messageId` from PATH as the ID to decode.

	msg, err := rt.db.ForwardMessage(req.ConversationID, messageId, ctx.UserID)
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
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageId := ps.ByName("messageId")
	if messageId == "" {
		http.Error(w, "Invalid Message ID", http.StatusBadRequest)
		return
	}

	err := rt.db.DeleteMessage(messageId, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to delete message")
		http.Error(w, "Not found or forbidden", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// commentMessage reacts to a message
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageId := ps.ByName("messageId")
	if messageId == "" {
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
	if ctx.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	reactionId := ps.ByName("reactionId")
	if reactionId == "" {
		http.Error(w, "Invalid Reaction ID", http.StatusBadRequest)
		return
	}

	err := rt.db.UncommentMessage(reactionId, ctx.UserID)
	if err != nil {
		ctx.Logger.WithError(err).Error("failed to uncomment")
		http.Error(w, "Not found or forbidden", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
