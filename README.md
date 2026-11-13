# BaseCore

BaseCore is an HTTP service written in Go. It includes a Chi-based API, a
SQLite database, bcrypt password hashing, and JWT-based authentication. It
provides basic CRUD endpoints for users, organizations, and projects that
belong to organizations.

## Features

- JWT authentication (access token + refresh token)
- Password storage with bcrypt
- WAL mode and foreign key constraints on SQLite
- Creating, listing, and deleting organizations
- Managing organization members (`owner` role only)
- Creating and deleting projects within an organization (`owner` role only)
- Storing the JWT secret in the system keyring

## Technologies

- [Go](https://go.dev/) 1.27+
- [Chi](https://github.com/go-chi/chi) — HTTP router
- [modernc.org/sqlite](https://modernc.org/sqlite) — SQLite driver without CGO
- [golang-jwt](https://github.com/golang-jwt/jwt) — JWT
- [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) — password hashing
- [go-keyring](https://github.com/zalando/go-keyring) — system keyring access

## Requirements

- Go 1.27 or later
- D-Bus and Secret Service for system keyring access on Linux
  (GNOME Keyring, KWallet, etc.)

## Installation

```sh
git clone <repository-url>
cd BaseCore
go mod download
```

Before running the application for the first time, store the JWT secret in the
system keyring:

```sh
go run . -jwt-secret
```

The command prompts for the secret interactively and stores it under the
`basecore` service with the name `jwt_secret_key`.

## Running

```sh
go run . -port 8080
```

Or build and run it:

```sh
go build -buildvcs=false -o basecore .
./basecore -port 8080
```

The application uses `~/.basecore` as its configuration directory. The SQLite
database is stored in `~/.basecore/basecore.db`, and its file permissions are
set to `0600`.

## API

All endpoints are under the `/api/v1` prefix. Protected endpoints require the
`Authorization: Bearer <token>` header.

| Method | Path | Authentication | Description |
| --- | --- | --- | --- |
| `POST` | `/register` | No | Register a new user |
| `POST` | `/login` | No | Log in; returns access and refresh tokens |
| `GET` | `/me` | Yes | Returns the user information from the token |
| `POST` | `/organizations` | Yes | Creates an organization |
| `GET` | `/organizations` | Yes | Lists the organizations the user belongs to |
| `DELETE` | `/organizations/{organizationID}` | Yes | Deletes an organization (`owner`) |
| `DELETE` | `/organizations/{organizationID}/members/{userID}` | Yes | Removes a member (`owner`) |
| `POST` | `/organizations/{organization_id}/projects` | Yes | Creates a project (`owner`) |
| `DELETE` | `/organizations/{organization_id}/projects/{project_id}` | Yes | Deletes a project (`owner`) |

### Example flow

User registration:

```sh
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"ada","password":"strong-password"}'
```

Login:

```sh
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"ada","password":"strong-password"}'
```

Response:

```json
{
  "token": "<access-token>",
  "refreshToken": "<refresh-token>",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

Creating an organization:

```sh
curl -X POST http://localhost:8080/api/v1/organizations \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Acme","description":"Acme Corporation"}'
```

Creating a project:

```sh
curl -X POST http://localhost:8080/api/v1/organizations/1/projects \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Website"}'
```

## Project Structure

```text
.
├── main.go                     # Application entry point
├── internal/
│   ├── api/
│   │   ├── router.go           # HTTP routes
│   │   ├── handlers/           # HTTP handlers (auth, organization, projects)
│   │   ├── middlewares/        # JWT validation middleware
│   │   └── models/             # Request/response and JWT claim types
│   ├── config/                 # Configuration and keyring secret management
│   ├── repository/             # Database operations
│   └── store/                  # SQLite connection and schema
├── AGENTS.md                   # Project notes for development agents
├── go.mod
└── go.sum
```

## Security Notes

- Passwords are stored as bcrypt hashes.
- The JWT secret is not kept in source code or on disk; it is stored in the
  system keyring.
- All SQL queries are executed with parameters.
- The database file is created with `0600` permissions.
- Authentication errors return generic messages so they do not reveal whether
  an account exists.
- Request bodies are limited to 1 MiB with `http.MaxBytesReader`.

## License

This project is licensed under the [MIT License](LICENSE).
