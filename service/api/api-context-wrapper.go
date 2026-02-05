package api

import (
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/kubilaykoccc/Wasa/service/api/reqcontext"
	"github.com/sirupsen/logrus"
)

// httpRouterHandler is the signature for functions that accepts a reqcontext.RequestContext in addition to those
// required by the httprouter package.
type httpRouterHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext)

// wrap parses the request and adds a reqcontext.RequestContext instance related to the request.
func (rt *_router) wrap(fn httpRouterHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		reqUUID, err := uuid.NewV4()
		if err != nil {
			rt.baseLogger.WithError(err).Error("can't generate a request UUID")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var ctx = reqcontext.RequestContext{
			ReqUUID: reqUUID,
		}

		// Create a request-specific logger
		ctx.Logger = rt.baseLogger.WithFields(logrus.Fields{
			"reqid":     ctx.ReqUUID.String(),
			"remote-ip": r.RemoteAddr,
		})

		// Extract authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Expecting "Bearer <token>"
			const prefix = "Bearer "
			if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
				tokenStr := authHeader[len(prefix):]
				id, err := strconv.ParseUint(tokenStr, 10, 64)
				if err == nil {
					ctx.UserID = id
				}
			}
		}

		// Call the next handler in chain (usually, the handler function for the path)
		fn(w, r, ps, ctx)
	}
}
