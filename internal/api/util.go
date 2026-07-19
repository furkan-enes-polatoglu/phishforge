package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func uuidMust(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

// urlUUID parses a path parameter as a UUID, writing a 400 on failure.
func urlUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}
