package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.GET("/", rt.getHelloWorld)
	rt.router.GET("/context", rt.wrap(rt.getContextReply))

	// Login
	rt.router.POST("/session", rt.wrap(rt.doLogin))

	// Users
	rt.router.PUT("/users/me/name", rt.wrap(rt.setMyUserName))
	rt.router.PUT("/users/me/photo", rt.wrap(rt.setMyPhoto))
	rt.router.GET("/users", rt.wrap(rt.searchUsers))

	// Conversations
	rt.router.GET("/conversations", rt.wrap(rt.getMyConversations))
	rt.router.POST("/conversations", rt.wrap(rt.createConversation))
	rt.router.GET("/conversations/:conversationId", rt.wrap(rt.getConversation))
	rt.router.PUT("/conversations/:conversationId/name", rt.wrap(rt.setGroupName))
	rt.router.PUT("/conversations/:conversationId/photo", rt.wrap(rt.setGroupPhoto))
	rt.router.PUT("/conversations/:conversationId/members/:userId", rt.wrap(rt.addToGroup))
	rt.router.DELETE("/conversations/:conversationId/members/me", rt.wrap(rt.leaveGroup))

	// Messages
	rt.router.POST("/conversations/:conversationId/messages", rt.wrap(rt.sendMessage))
	// rt.router.POST("/conversations/:conversationId/messages/forwarded", rt.wrap(rt.forwardMessage)) // Handled by wildcard
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId", rt.wrap(rt.deleteMessage))

	// Reactions and Forwarding (combined to avoid httprouter conflict)
	rt.router.POST("/conversations/:conversationId/messages/*action", rt.wrap(rt.handleMessageAction))
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId/reactions/:reactionId", rt.wrap(rt.uncommentMessage))

	// Special routes
	rt.router.GET("/liveness", rt.liveness)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		rt.router.ServeHTTP(w, r)
	})
}
