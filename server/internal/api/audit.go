package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/alvarorg14/openlicensd/server/internal/auth"
	"github.com/alvarorg14/openlicensd/server/internal/logging"
	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type auditRouteInfo struct {
	Action       string
	ResourceType string
	IDParam      string // chi URL param name for resource ID, empty for creates
}

// auditRoutes maps "METHOD routePattern" to audit metadata. Route patterns must match
// chi.RouteContext(ctx).RoutePattern() for authenticated mutating handlers.
var auditRoutes = map[string]auditRouteInfo{
	"POST /api/v1/auth/logout":                          {Action: "auth.logout", ResourceType: "session"},
	"POST /api/v1/auth/password":                        {Action: "auth.password_change", ResourceType: "user", IDParam: "self"},
	"POST /api/v1/licenses":                             {Action: "license.create", ResourceType: "license"},
	"PATCH /api/v1/licenses/{id}":                       {Action: "license.update", ResourceType: "license", IDParam: "id"},
	"DELETE /api/v1/licenses/{id}":                      {Action: "license.delete", ResourceType: "license", IDParam: "id"},
	"PATCH /api/v1/licenses/{id}/revoke":                {Action: "license.revoke", ResourceType: "license", IDParam: "id"},
	"PATCH /api/v1/licenses/{id}/unrevoke":              {Action: "license.unrevoke", ResourceType: "license", IDParam: "id"},
	"PATCH /api/v1/licenses/{id}/machines/{machineId}":  {Action: "machine.update", ResourceType: "machine", IDParam: "machineId"},
	"DELETE /api/v1/licenses/{id}/machines/{machineId}": {Action: "machine.release", ResourceType: "machine", IDParam: "machineId"},
	"POST /api/v1/products":                             {Action: "product.create", ResourceType: "product"},
	"PATCH /api/v1/products/{id}":                       {Action: "product.update", ResourceType: "product", IDParam: "id"},
	"DELETE /api/v1/products/{id}":                      {Action: "product.delete", ResourceType: "product", IDParam: "id"},
	"POST /api/v1/policies":                             {Action: "policy.create", ResourceType: "policy"},
	"PATCH /api/v1/policies/{id}":                       {Action: "policy.update", ResourceType: "policy", IDParam: "id"},
	"DELETE /api/v1/policies/{id}":                      {Action: "policy.delete", ResourceType: "policy", IDParam: "id"},
	"POST /api/v1/users":                                {Action: "user.create", ResourceType: "user"},
	"PATCH /api/v1/users/{id}":                          {Action: "user.update", ResourceType: "user", IDParam: "id"},
	"PATCH /api/v1/users/{id}/password":                 {Action: "user.password_set", ResourceType: "user", IDParam: "id"},
	"PATCH /api/v1/users/{id}/disable":                  {Action: "user.disable", ResourceType: "user", IDParam: "id"},
	"PATCH /api/v1/users/{id}/enable":                   {Action: "user.enable", ResourceType: "user", IDParam: "id"},
	"DELETE /api/v1/users/{id}":                         {Action: "user.delete", ResourceType: "user", IDParam: "id"},
	"POST /api/v1/api-tokens":                           {Action: "api_token.create", ResourceType: "api_token"},
	"PATCH /api/v1/api-tokens/{id}/revoke":              {Action: "api_token.revoke", ResourceType: "api_token", IDParam: "id"},
	"DELETE /api/v1/api-tokens/{id}":                    {Action: "api_token.delete", ResourceType: "api_token", IDParam: "id"},
}

type auditRecorderKey struct{}

type auditRecorder struct {
	resourceID    *uuid.UUID
	resourceLabel *string
	metadata      json.RawMessage
}

func auditRecorderFromContext(ctx context.Context) *auditRecorder {
	rec, _ := ctx.Value(auditRecorderKey{}).(*auditRecorder)
	if rec == nil {
		rec = &auditRecorder{}
	}
	return rec
}

// auditResource annotates the in-flight audit record with the target resource ID and label.
// Handlers should call this after a successful mutation when the ID is not in the URL.
func auditResource(ctx context.Context, id uuid.UUID, label string) {
	rec := auditRecorderFromContext(ctx)
	idCopy := id
	rec.resourceID = &idCopy
	if label != "" {
		rec.resourceLabel = &label
	}
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func auditRouteKey(method, pattern string) string {
	return method + " " + pattern
}

func (s *Server) auditMutations(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMutatingMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		routeCtx := chi.RouteContext(r.Context())
		if routeCtx == nil {
			next.ServeHTTP(w, r)
			return
		}

		pattern := routeCtx.RoutePattern()
		key := auditRouteKey(r.Method, pattern)
		info, tracked := auditRoutes[key]
		if !tracked {
			next.ServeHTTP(w, r)
			return
		}

		rec := &auditRecorder{}
		ctx := context.WithValue(r.Context(), auditRecorderKey{}, rec)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r.WithContext(ctx))

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		if status < 200 || status >= 300 {
			return
		}

		principal, ok := auth.PrincipalFromContext(ctx)
		if !ok {
			return
		}

		var resourceID *uuid.UUID
		if rec.resourceID != nil {
			resourceID = rec.resourceID
		} else if info.IDParam != "" && info.IDParam != "self" {
			if raw := chi.URLParam(r, info.IDParam); raw != "" {
				if id, err := uuid.Parse(raw); err == nil {
					resourceID = &id
				}
			}
		} else if info.IDParam == "self" {
			if principal.UserID != uuid.Nil {
				id := principal.UserID
				resourceID = &id
			}
		}

		var actorUserID, actorTokenID *uuid.UUID
		var actorEmail, actorTokenPrefix *string
		if principal.AuthMethod == auth.AuthMethodAPIToken {
			if principal.TokenID != uuid.Nil {
				id := principal.TokenID
				actorTokenID = &id
			}
			if principal.TokenPrefix != "" {
				prefix := principal.TokenPrefix
				actorTokenPrefix = &prefix
			}
		} else {
			if principal.UserID != uuid.Nil {
				id := principal.UserID
				actorUserID = &id
			}
			if principal.Email != "" {
				email := principal.Email
				actorEmail = &email
			}
		}

		var clientIP, userAgent, requestID *string
		if ip := s.clientIP.From(r); ip != "" {
			clientIP = &ip
		}
		if ua := r.UserAgent(); ua != "" {
			userAgent = &ua
		}
		if rid := middleware.GetReqID(ctx); rid != "" {
			requestID = &rid
		}

		event := store.AuditEvent{
			Action:           info.Action,
			ResourceType:     info.ResourceType,
			ResourceID:       resourceID,
			ResourceLabel:    rec.resourceLabel,
			ActorUserID:      actorUserID,
			ActorTokenID:     actorTokenID,
			ActorName:        principal.Name,
			ActorEmail:       actorEmail,
			ActorTokenPrefix: actorTokenPrefix,
			ActorRole:        string(principal.Role),
			AuthMethod:       string(principal.AuthMethod),
			ClientIP:         clientIP,
			UserAgent:        userAgent,
			RequestID:        requestID,
			RequestMethod:    r.Method,
			RequestPath:      r.URL.Path,
			ResponseStatus:   status,
			Metadata:         rec.metadata,
		}

		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		if err := s.store.CreateAuditEvent(writeCtx, event); err != nil {
			logging.FromContext(ctx).Error("failed to write audit event",
				slog.String("action", info.Action),
				slog.Any("err", err),
			)
		}
	})
}
