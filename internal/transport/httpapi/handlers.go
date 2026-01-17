package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"profile-aggregator/internal/usecase"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func ProfileHandler(uc usecase.ProfileUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := ""
		if r.URL.Path != "/profile" {
			parts := splitPath(r.URL.Path)
			if len(parts) == 2 && parts[0] == "profile" && parts[1] != "" {
				idStr = parts[1]
			}
		}
		if idStr == "" {
			idStr = r.URL.Query().Get("id")
		}
		if idStr == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing id"})
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid uuid"})
			return
		}

		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			clientID = "default"
		}

		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		profile, err := uc.GetProfile(ctx, clientID, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal error"})
			return
		}

		resp := map[string]any{
			"id": profile.ID.String(),
		}
		if profile.Email != "" {
			resp["email"] = profile.Email
		}
		if profile.Name != "" {
			resp["name"] = profile.Name
		}
		if profile.AvatarURL != "" {
			resp["avatar_url"] = profile.AvatarURL
		}
		for k, v := range profile.Fields {
			resp[k] = v
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func splitPath(p string) []string {
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	if p == "" {
		return nil
	}
	parts := []string{}
	cur := ""
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			if cur != "" {
				parts = append(parts, cur)
				cur = ""
			}
			continue
		}
		cur += string(p[i])
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}
