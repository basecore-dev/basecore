package projects

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/basecore-dev/basecore/internal/api/middlewares"
	"github.com/basecore-dev/basecore/internal/api/models"
	"github.com/basecore-dev/basecore/internal/repository"
	"github.com/basecore-dev/basecore/internal/store"
	"github.com/go-chi/chi/v5"
)

func CreateNewProjectHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "method_not_allowed",
		})
		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "database_not_connected",
		})
		return
	}

	claims, ok := middlewares.GetClaims(r)

	if !ok {
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "unauthorized",
		})

		return
	}

	organizationID, err := strconv.Atoi(
		chi.URLParam(r, "organization_id"),
	)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_organization_id",
		})

		return
	}

	var existingOrganizationID int64

	err = store.DB.QueryRow(
		"SELECT id FROM organizations WHERE id = ?",
		organizationID,
	).Scan(&existingOrganizationID)

	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "organization_not_found",
		})

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "organization_lookup_failed",
		})

		return
	}

	isOwner, err := repository.IsOrganizationOwner(
		int64(organizationID),
		int64(claims.UserID),
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "organization_ownership_check_failed",
		})

		return
	}

	if !isOwner {
		w.WriteHeader(http.StatusForbidden)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "not_organization_owner",
		})

		return
	}

	var req models.CreateNewProject

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_json",
		})

		return
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_json",
		})

		return
	}

	req.Title = strings.TrimSpace(req.Title)

	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "title_is_required",
		})

		return
	}

	projectID, err := repository.InsertNewProject(
		int64(organizationID),
		req.Title,
	)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProjectAlreadyExists):
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "project_already_exists",
			})
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "project_creation_failed",
			})
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":    "project_created",
		"project_id": projectID,
	})
}

// DeleteProjectHandler deletes a project when the authenticated user is the
// organization's owner.
func DeleteProjectHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "method_not_allowed",
		})
		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "database_not_connected",
		})
		return
	}

	claims, ok := middlewares.GetClaims(r)

	if !ok {
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "unauthorized",
		})

		return
	}

	organizationID, err := strconv.Atoi(
		chi.URLParam(r, "organization_id"),
	)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_organization_id",
		})

		return
	}

	projectID, err := strconv.Atoi(
		chi.URLParam(r, "project_id"),
	)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_project_id",
		})

		return
	}

	isOwner, err := repository.IsOrganizationOwner(
		int64(organizationID),
		int64(claims.UserID),
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "organization_ownership_check_failed",
		})

		return
	}

	if !isOwner {
		w.WriteHeader(http.StatusForbidden)

		json.NewEncoder(w).Encode(map[string]any{
			"error": "not_organization_owner",
		})

		return
	}

	err = repository.DeleteProject(
		int64(organizationID),
		int64(projectID),
	)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProjectNotFound):
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "project_not_found",
			})
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "project_deletion_failed",
			})
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "project_deleted",
	})
}
