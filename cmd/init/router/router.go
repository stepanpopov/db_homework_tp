package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stepanpopov/db_homework_tp/internal/delivery"
	"github.com/stepanpopov/db_homework_tp/internal/repo"
)


// InitRouter describes all app's endpoints and their handlers
func Init(db *pgxpool.Pool) *chi.Mux {

	repo := repo.NewRepo(db)
	h := delivery.NewHandler(repo)

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Route("/user", func(r chi.Router) {
			r.Route("/{nickname}", func(r chi.Router) {
				r.Post("/create", h.UserCreate)
				r.Get("/profile", h.UserGet)
				r.Post("/profile", h.UserUpdate)	
			})
		})

		r.Route("/forum", func(r chi.Router) {
			r.Post("/create", h.ForumCreate)
			r.Route("/{slug}", func(r chi.Router) {
				r.Get("/details", h.ForumGetBySlug)
				r.Get("/users", h.ForumGetUsers)
				r.Post("/create", h.ThreadCreate)
			})
		})

		r.Route("/service", func(r chi.Router) {
			r.Post("/clear", h.ServiceClear)
			r.Get("/status", h.ServiceStatus)
		})
	})

	
	

	return r
}
