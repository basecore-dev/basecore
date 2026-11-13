package models

import "time"

// CreateNewOrgReq contains the title and description submitted when creating
// an organization.
type CreateNewOrgReq struct {
	// Title is the organization's display name.
	Title string `json:"title"`
	// Description describes the organization.
	Description string `json:"description"`
}

// OrganizationResponse describes an organization and the requesting user's
// membership in it.
type OrganizationResponse struct {
	// ID is the organization's identifier.
	ID int64 `json:"id"`
	// Title is the organization's display name.
	Title string `json:"title"`
	// Description describes the organization.
	Description string `json:"description"`
	// Role is the requesting user's role in the organization.
	Role string `json:"role"`
	// JoinedAt is when the requesting user joined the organization.
	JoinedAt time.Time `json:"joined_at"`
	// CreatedAt is when the organization was created.
	CreatedAt time.Time `json:"created_at"`
}
