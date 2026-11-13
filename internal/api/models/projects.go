package models

// CreateNewProject contains the title submitted when creating a project.
type CreateNewProject struct {
	// Title is the project's display name.
	Title string `json:"title"`
}

// CreateNewAPIKeyReq contains the title and permissions submitted when creating
// an API key.
type CreateNewAPIKeyReq struct {
	// Title is the API key's display name.
	Title string `json:"title"`
	// Permissions is one of "all", "view", "delete", or "insert".
	Permissions string `json:"permissions"`
}

// CreateSQLiteDBReq contains the title submitted when creating a project's
// SQLite database.
type CreateSQLiteDBReq struct {
	// Title is the database name; the file is stored as <title>.sqlite.
	Title string `json:"title"`
}
