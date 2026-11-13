# Agent Instructions

## Project

BaseCore is a Go service. The HTTP API is built with Chi, SQLite access uses
`database/sql` and `modernc.org/sqlite`, and authentication uses bcrypt and JWT.

## Structure

- `internal/api/router.go` registers API routes.
- `internal/api/handlers/` contains HTTP handlers.
- `internal/api/middlewares/` contains HTTP middleware.
- `internal/api/models/` contains API request and JWT claim types.
- `internal/repository/` contains database operations.
- `internal/store/` manages the SQLite connection and schema.
- `internal/config/` manages local configuration and keyring secrets.

## Conventions

- Keep code inside `internal/` unless it must be exposed outside this module.
- Use parameterized SQL queries; never concatenate user input into SQL.
- Store passwords as bcrypt hashes and keep JWT secrets in the system keyring.
- Return generic authentication errors so responses do not reveal whether an
  account exists.
- Keep HTTP handlers explicit about status codes and JSON response bodies.
- Add GoDoc comments to exported declarations and run `gofmt` on changed Go
  files.

## Build

Build the module with:

```sh
go build -buildvcs=false ./...
```
