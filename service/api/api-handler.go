package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handles APIs registered here
func (rt *_router) Handler() http.Handler {
	// Login endpoints (no auth required)
	h := rt
	h.router.POST("/session", h.wrap(h.doLogin))

	// User endpoints
	h.router.PUT("/users/me/name", h.wrap(h.setMyUserName))
	h.router.PUT("/users/me/photo", h.wrap(h.setMyPhoto))

	// Conversation endpoints
	h.router.GET("/conversations", h.wrap(h.getMyConversations))
	h.router.POST("/start-conversation", h.wrap(h.startConversation)) // Optional endpoint for UI
	h.router.GET("/conversations/:conversationId", h.wrap(h.getConversation))
	h.router.POST("/conversations/:conversationId/messages", h.wrap(h.sendMessage))

	// Message endpoints
	h.router.DELETE("/messages/:messageId", h.wrap(h.deleteMessage))
	h.router.POST("/messages/:messageId/forward", h.wrap(h.forwardMessage))
	h.router.POST("/messages/:messageId/reactions", h.wrap(h.commentMessage))
	h.router.DELETE("/messages/:messageId/reactions/:reactionId", h.wrap(h.uncommentMessage))

	// Group endpoints
	h.router.POST("/groups/:groupId/leave", h.wrap(h.leaveGroup))
	h.router.POST("/groups/:groupId/members", h.wrap(h.addToGroup))
	h.router.PUT("/groups/:groupId/name", h.wrap(h.setGroupName))
	h.router.PUT("/groups/:groupId/photo", h.wrap(h.setGroupPhoto))

	// endpoints for the frontend
	h.router.GET("/users/search", h.wrap(h.searchUsers))
	h.router.ServeFiles("/photos/*filepath", http.Dir("data/photos"))

	// Special endpoints
	h.router.GET("/liveness", h.liveness)

	return h.router
}
