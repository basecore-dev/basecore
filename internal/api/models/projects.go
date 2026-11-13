package models

// CreateNewProject contains the title submitted when creating a project.
type CreateNewProject struct {
	// Title is the project's display name.
	Title string `json:"title"`
}
