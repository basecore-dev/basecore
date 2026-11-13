package organization

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/basecore-dev/basecore/internal/api/middlewares"
	"github.com/basecore-dev/basecore/internal/api/models"
	"github.com/basecore-dev/basecore/internal/repository"
	"github.com/basecore-dev/basecore/internal/store"
	"github.com/go-chi/chi/v5"
)

const (
	minOrganizationTitleLength       = 3
	maxOrganizationTitleLength       = 100
	minOrganizationDescriptionLength = 10
	maxOrganizationDescriptionLength = 1000
)

// CreateNewOrganization handles organization creation requests.
func CreateNewOrganization(
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

	var req models.CreateNewOrgReq

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
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "title_is_required",
		})
		return
	}

	titleLength := utf8.RuneCountInString(req.Title)
	if titleLength < minOrganizationTitleLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "title_too_short",
		})
		return
	}
	if titleLength > maxOrganizationTitleLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "title_too_long",
		})
		return
	}
	if strings.ContainsFunc(req.Title, unicode.IsControl) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "title_contains_control_characters",
		})
		return
	}

	if req.Description == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "description_is_required",
		})
		return
	}

	descriptionLength := utf8.RuneCountInString(req.Description)
	if descriptionLength < minOrganizationDescriptionLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "description_too_short",
		})
		return
	}
	if descriptionLength > maxOrganizationDescriptionLength {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "description_too_long",
		})
		return
	}
	if strings.ContainsFunc(req.Description, func(r rune) bool {
		if r == '\n' || r == '\r' || r == '\t' {
			return false
		}

		return unicode.IsControl(r)
	}) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "description_contains_control_characters",
		})
		return
	}

	organizationID, err := repository.CreateOrganization(
		int64(claims.UserID),
		req.Title,
		req.Description,
	)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organization_creation_failed",
			})
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":         "organization_created",
		"organization_id": organizationID,
	})
}

// GetMyOrganizations returns the organizations the authenticated user is a
// member of.
func GetMyOrganizations(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "method_not_allowed",
		})
		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
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

	organizations, err := repository.GetUserOrganizations(int64(claims.UserID))

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organizations_fetch_failed",
			})
		}

		return
	}

	response := make([]models.OrganizationResponse, 0, len(organizations))

	for _, organization := range organizations {
		response = append(response, models.OrganizationResponse{
			ID:          organization.ID,
			Title:       organization.Title,
			Description: organization.Description,
			Role:        organization.Role,
			JoinedAt:    organization.JoinedAt,
			CreatedAt:   organization.CreatedAt,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"organizations": response,
	})
}

// DeleteOrganization deletes the organization identified by the URL parameter
// when the authenticated user is its owner.
func DeleteOrganization(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "method_not_allowed",
		})
		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
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

	organizationID, err := strconv.ParseInt(
		chi.URLParam(r, "organizationID"),
		10,
		64,
	)

	if err != nil || organizationID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_organization_id",
		})
		return
	}

	err = repository.DeleteOrganization(organizationID, int64(claims.UserID))

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organization_not_found",
			})
		case errors.Is(err, repository.ErrNotOrganizationOwner):
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "not_organization_owner",
			})
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organization_deletion_failed",
			})
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "organization_deleted",
	})
}

// RemoveOrganizationMember removes the user identified by the URL parameter
// from the organization when the authenticated user is its owner.
func RemoveOrganizationMember(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "method_not_allowed",
		})
		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
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

	organizationID, err := strconv.ParseInt(
		chi.URLParam(r, "organizationID"),
		10,
		64,
	)

	if err != nil || organizationID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_organization_id",
		})
		return
	}

	memberID, err := strconv.ParseInt(
		chi.URLParam(r, "userID"),
		10,
		64,
	)

	if err != nil || memberID < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_user_id",
		})
		return
	}

	err = repository.RemoveOrganizationMember(
		organizationID,
		int64(claims.UserID),
		memberID,
	)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organization_not_found",
			})
		case errors.Is(err, repository.ErrNotOrganizationOwner):
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "not_organization_owner",
			})
		case errors.Is(err, repository.ErrOrganizationMemberNotFound):
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organization_member_not_found",
			})
		case errors.Is(err, repository.ErrCannotRemoveOrganizationOwner):
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "cannot_remove_organization_owner",
			})
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "organization_member_removal_failed",
			})
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "organization_member_removed",
	})
}
