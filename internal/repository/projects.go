package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/basecore-dev/basecore/internal/store"
)

// ErrProjectAlreadyExists is returned when an organization already has a
// project with the same title.
var ErrProjectAlreadyExists = errors.New("repository: project already exists")

// ErrProjectNotFound is returned when a project does not exist in an
// organization.
var ErrProjectNotFound = errors.New("repository: project not found")

// Permissions accepted for an API key.
const (
	// APIKeyPermissionAll grants every operation.
	APIKeyPermissionAll = "all"
	// APIKeyPermissionView grants read access.
	APIKeyPermissionView = "view"
	// APIKeyPermissionDelete grants delete access.
	APIKeyPermissionDelete = "delete"
	// APIKeyPermissionInsert grants insert access.
	APIKeyPermissionInsert = "insert"
)

// ErrInvalidAPIKeyPermission is returned when an API key permission is not one
// of the supported values.
var ErrInvalidAPIKeyPermission = errors.New("repository: invalid API key permission")

// ErrAPIKeyAlreadyExists is returned when a project already has an API key with
// the same title.
var ErrAPIKeyAlreadyExists = errors.New("repository: API key already exists")

// InsertNewProject stores a project titled title for organizationID and returns
// the new project's ID.
// It returns ErrProjectAlreadyExists when the organization already has a project
// with the same title.
func InsertNewProject(organizationID int64, title string) (int64, error) {
	if store.DB == nil {
		return 0, ErrDatabaseNotConnected
	}

	result, err := store.DB.Exec(
		"INSERT INTO projects (organization_id, title) VALUES (?, ?)",
		organizationID,
		title,
	)
	if err != nil {
		if store.IsUniqueError(err) {
			return 0, ErrProjectAlreadyExists
		}

		return 0, fmt.Errorf("repository: insert project: %w", err)
	}

	projectID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("repository: get project ID: %w", err)
	}

	return projectID, nil
}

// DeleteProject deletes projectID from organizationID.
// It returns ErrProjectNotFound when the organization has no project with that
// ID.
func DeleteProject(organizationID, projectID int64) error {
	if store.DB == nil {
		return ErrDatabaseNotConnected
	}

	result, err := store.DB.Exec(
		"DELETE FROM projects WHERE id = ? AND organization_id = ?",
		projectID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("repository: delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: get deleted project count: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProjectNotFound
	}

	return nil
}

// InsertAPIKey stores an API key hash, titled title, for projectID and returns
// the new key's ID. permissions must be one of APIKeyPermissionAll,
// APIKeyPermissionView, APIKeyPermissionDelete, or APIKeyPermissionInsert.
// It returns ErrInvalidAPIKeyPermission when permissions is not supported and
// ErrAPIKeyAlreadyExists when the project already has a key with the same
// title.
func InsertAPIKey(projectID int64, title, keyHash, permissions string) (int64, error) {
	if store.DB == nil {
		return 0, ErrDatabaseNotConnected
	}

	if !IsValidAPIKeyPermission(permissions) {
		return 0, ErrInvalidAPIKeyPermission
	}

	result, err := store.DB.Exec(
		"INSERT INTO api_keys (project_id, title, key_hash, permissions) VALUES (?, ?, ?, ?)",
		projectID,
		title,
		keyHash,
		permissions,
	)
	if err != nil {
		if store.IsUniqueError(err) {
			return 0, ErrAPIKeyAlreadyExists
		}

		return 0, fmt.Errorf("repository: insert API key: %w", err)
	}

	apiKeyID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("repository: get API key ID: %w", err)
	}

	return apiKeyID, nil
}

// GetOrganizationProjectIDs returns the IDs of the projects that belong to
// organizationID.
func GetOrganizationProjectIDs(organizationID int64) ([]int64, error) {
	if store.DB == nil {
		return nil, ErrDatabaseNotConnected
	}

	rows, err := store.DB.Query(
		"SELECT id FROM projects WHERE organization_id = ? ORDER BY id",
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: query organization projects: %w", err)
	}
	defer rows.Close()

	projectIDs := make([]int64, 0)

	for rows.Next() {
		var projectID int64

		if err := rows.Scan(&projectID); err != nil {
			return nil, fmt.Errorf("repository: scan organization project: %w", err)
		}

		projectIDs = append(projectIDs, projectID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate organization projects: %w", err)
	}

	return projectIDs, nil
}

// IsValidAPIKeyPermission reports whether permission is a supported API key
// permission.
func IsValidAPIKeyPermission(permission string) bool {
	switch permission {
	case APIKeyPermissionAll,
		APIKeyPermissionView,
		APIKeyPermissionDelete,
		APIKeyPermissionInsert:
		return true
	default:
		return false
	}
}

// ProjectExists reports whether projectID belongs to organizationID.
func ProjectExists(organizationID, projectID int64) (bool, error) {
	if store.DB == nil {
		return false, ErrDatabaseNotConnected
	}

	var id int64

	err := store.DB.QueryRow(
		"SELECT id FROM projects WHERE id = ? AND organization_id = ?",
		projectID,
		organizationID,
	).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("repository: check project: %w", err)
	}

	return true, nil
}

// IsOrganizationOwner reports whether userID is the owner of organizationID.
// It returns false without an error when the user is not a member.
func IsOrganizationOwner(organizationID, userID int64) (bool, error) {
	if store.DB == nil {
		return false, ErrDatabaseNotConnected
	}

	var role string

	err := store.DB.QueryRow(
		"SELECT role FROM organizations_members WHERE organization_id = ? AND user_id = ?",
		organizationID,
		userID,
	).Scan(&role)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("repository: check organization owner: %w", err)
	}

	return role == organizationRoleOwner, nil
}
