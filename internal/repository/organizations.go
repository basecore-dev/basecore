package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/basecore-dev/basecore/internal/store"
)

const organizationRoleOwner = "owner"

var (
	// ErrOrganizationNotFound is returned when an organization does not exist.
	ErrOrganizationNotFound = errors.New("repository: organization not found")
	// ErrNotOrganizationOwner is returned when a user is not the organization's
	// owner.
	ErrNotOrganizationOwner = errors.New("repository: user is not an organization owner")
	// ErrOrganizationMemberNotFound is returned when a user is not a member of
	// the organization.
	ErrOrganizationMemberNotFound = errors.New("repository: organization member not found")
	// ErrCannotRemoveOrganizationOwner is returned when the membership being
	// removed belongs to the organization's owner.
	ErrCannotRemoveOrganizationOwner = errors.New("repository: cannot remove organization owner")
)

// Organization is an organization together with the current user's membership
// details in it.
type Organization struct {
	// ID is the organization's identifier.
	ID int64
	// Title is the organization's display name.
	Title string
	// Description describes the organization.
	Description string
	// Role is the current user's role in the organization.
	Role string
	// JoinedAt is when the current user joined the organization.
	JoinedAt time.Time
	// CreatedAt is when the organization was created.
	CreatedAt time.Time
}

// CreateOrganization creates an organization and registers creatorUserID as its
// owner in a single transaction. It returns the new organization's ID.
func CreateOrganization(creatorUserID int64, title, description string) (int64, error) {
	if store.DB == nil {
		return 0, ErrDatabaseNotConnected
	}

	tx, err := store.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("repository: begin create organization transaction: %w", err)
	}

	// Roll back if any step fails. After a successful commit this is a no-op
	// that returns sql.ErrTxDone.
	defer tx.Rollback()

	result, err := tx.Exec(
		"INSERT INTO organizations (title, description) VALUES (?, ?)",
		title,
		description,
	)
	if err != nil {
		return 0, fmt.Errorf("repository: insert organization: %w", err)
	}

	organizationID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("repository: get organization ID: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO organizations_members (organization_id, user_id, role) VALUES (?, ?, ?)",
		organizationID,
		creatorUserID,
		organizationRoleOwner,
	); err != nil {
		return 0, fmt.Errorf("repository: add organization owner: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("repository: commit create organization transaction: %w", err)
	}

	return organizationID, nil
}

// GetUserOrganizations returns every organization userID is a member of,
// newest first. The returned slice is empty when the user has no memberships.
func GetUserOrganizations(userID int64) ([]Organization, error) {
	if store.DB == nil {
		return nil, ErrDatabaseNotConnected
	}

	rows, err := store.DB.Query(
		`SELECT o.id, o.title, o.description, m.role, m.joined_at, o.created_at
		FROM organizations_members m
		JOIN organizations o ON o.id = m.organization_id
		WHERE m.user_id = ?
		ORDER BY o.created_at DESC, o.id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: query user organizations: %w", err)
	}
	defer rows.Close()

	organizations := make([]Organization, 0)

	for rows.Next() {
		var organization Organization

		if err := rows.Scan(
			&organization.ID,
			&organization.Title,
			&organization.Description,
			&organization.Role,
			&organization.JoinedAt,
			&organization.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan user organization: %w", err)
		}

		organizations = append(organizations, organization)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate user organizations: %w", err)
	}

	return organizations, nil
}

// DeleteOrganization deletes organizationID when userID is its owner. Memberships
// are removed by the foreign key cascade.
// It returns ErrOrganizationNotFound when the organization does not exist and
// ErrNotOrganizationOwner when userID is not its owner.
func DeleteOrganization(organizationID, userID int64) error {
	if store.DB == nil {
		return ErrDatabaseNotConnected
	}

	tx, err := store.DB.Begin()
	if err != nil {
		return fmt.Errorf("repository: begin delete organization transaction: %w", err)
	}

	// Roll back if any step fails. After a successful commit this is a no-op
	// that returns sql.ErrTxDone.
	defer tx.Rollback()

	role, err := organizationMemberRole(tx, organizationID, userID)

	if errors.Is(err, ErrOrganizationMemberNotFound) {
		if err := organizationExists(tx, organizationID); err != nil {
			return err
		}

		return ErrNotOrganizationOwner
	}
	if err != nil {
		return err
	}

	if role != organizationRoleOwner {
		return ErrNotOrganizationOwner
	}

	if _, err := tx.Exec(
		"DELETE FROM organizations WHERE id = ?",
		organizationID,
	); err != nil {
		return fmt.Errorf("repository: delete organization: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit delete organization transaction: %w", err)
	}

	return nil
}

// organizationExists returns ErrOrganizationNotFound when organizationID is not
// stored, and nil otherwise.
func organizationExists(tx *sql.Tx, organizationID int64) error {
	var id int64

	err := tx.QueryRow(
		"SELECT id FROM organizations WHERE id = ?",
		organizationID,
	).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrOrganizationNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: find organization: %w", err)
	}

	return nil
}

// RemoveOrganizationMember removes memberUserID from organizationID when
// requesterUserID is the organization's owner.
// It returns ErrOrganizationNotFound when the organization does not exist,
// ErrNotOrganizationOwner when requesterUserID is not its owner,
// ErrOrganizationMemberNotFound when memberUserID is not a member, and
// ErrCannotRemoveOrganizationOwner when memberUserID is its owner.
func RemoveOrganizationMember(organizationID, requesterUserID, memberUserID int64) error {
	if store.DB == nil {
		return ErrDatabaseNotConnected
	}

	tx, err := store.DB.Begin()
	if err != nil {
		return fmt.Errorf("repository: begin remove organization member transaction: %w", err)
	}

	// Roll back if any step fails. After a successful commit this is a no-op
	// that returns sql.ErrTxDone.
	defer tx.Rollback()

	requesterRole, err := organizationMemberRole(tx, organizationID, requesterUserID)

	if errors.Is(err, ErrOrganizationMemberNotFound) {
		if err := organizationExists(tx, organizationID); err != nil {
			return err
		}

		return ErrNotOrganizationOwner
	}
	if err != nil {
		return err
	}

	if requesterRole != organizationRoleOwner {
		return ErrNotOrganizationOwner
	}

	memberRole, err := organizationMemberRole(tx, organizationID, memberUserID)
	if err != nil {
		return err
	}

	if memberRole == organizationRoleOwner {
		return ErrCannotRemoveOrganizationOwner
	}

	if _, err := tx.Exec(
		"DELETE FROM organizations_members WHERE organization_id = ? AND user_id = ?",
		organizationID,
		memberUserID,
	); err != nil {
		return fmt.Errorf("repository: delete organization member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit remove organization member transaction: %w", err)
	}

	return nil
}

// organizationMemberRole returns userID's role in organizationID, or
// ErrOrganizationMemberNotFound when the user is not a member.
func organizationMemberRole(tx *sql.Tx, organizationID, userID int64) (string, error) {
	var role string

	err := tx.QueryRow(
		"SELECT role FROM organizations_members WHERE organization_id = ? AND user_id = ?",
		organizationID,
		userID,
	).Scan(&role)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrOrganizationMemberNotFound
	}
	if err != nil {
		return "", fmt.Errorf("repository: find organization membership: %w", err)
	}

	return role, nil
}
