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
