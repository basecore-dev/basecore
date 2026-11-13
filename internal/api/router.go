// Package api configures the HTTP routes exposed by the application.
package api

import (
	"github.com/basecore-dev/basecore/internal/api/handlers/auth"
	"github.com/basecore-dev/basecore/internal/api/handlers/organization"
	"github.com/basecore-dev/basecore/internal/api/handlers/projects"
	"github.com/basecore-dev/basecore/internal/api/middlewares"
	"github.com/go-chi/chi/v5"
)

// Router creates the application's HTTP router and registers API endpoints.
func Router() *chi.Mux {
	router := chi.NewRouter()

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", auth.RegisterHandler)
		r.Post("/login", auth.LoginHandler)
		r.With(middlewares.AuthMiddleware).Get("/me", auth.GetMe)

		r.With(
			middlewares.AuthMiddleware,
		).Post(
			"/organizations", organization.CreateNewOrganization,
		)
		r.With(
			middlewares.AuthMiddleware,
		).Get(
			"/organizations", organization.GetMyOrganizations,
		)
		r.With(
			middlewares.AuthMiddleware,
		).Delete(
			"/organizations/{organizationID}", organization.DeleteOrganization,
		)
		r.With(
			middlewares.AuthMiddleware,
		).Delete(
			"/organizations/{organizationID}/members/{userID}", organization.RemoveOrganizationMember,
		)
		r.With(
			middlewares.AuthMiddleware,
		).Post(
			"/organizations/{organization_id}/projects", projects.CreateNewProjectHandler,
		)
		r.With(
			middlewares.AuthMiddleware,
		).Delete(
			"/organizations/{organization_id}/projects/{project_id}", projects.DeleteProjectHandler,
		)
	})
	return router
}
