package delivery

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mailru/easyjson"

	"github.com/stepanpopov/db_homework_tp/internal/models"
	"github.com/stepanpopov/db_homework_tp/internal/repo"
	"github.com/stepanpopov/db_homework_tp/internal/utils"
)

type Handler struct {
	repo *repo.Repo
}

func NewHandler(repo *repo.Repo) *Handler {
	return &Handler{
		repo: repo,
	}
}

// -- USER --

func (h *Handler) UserCreate(w http.ResponseWriter, r *http.Request) {
	u := models.User{}
	easyjson.UnmarshalFromReader(r.Body, &u)
	u.Nickname = chi.URLParam(r, "nickname")

	if err := h.repo.UserCreate(u); err != nil {
		users := h.repo.UserGetByNicknameOrEmail(u.Nickname, u.Email)
		utils.Response(w, 409, users)
		return
	}

	utils.Response(w, 201, u)
}

func (h *Handler) UserGet(w http.ResponseWriter, r *http.Request) {
	nickname := chi.URLParam(r, "nickname")
	u, err := h.repo.UserGet(nickname)
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}
	utils.Response(w, 200, u)
}

func (h *Handler) UserUpdate(w http.ResponseWriter, r *http.Request) {
	u := models.User{}
	easyjson.UnmarshalFromReader(r.Body, &u)
	u.Nickname = chi.URLParam(r, "nickname")

	newU, err := h.repo.UserUpdate(u)
	if err != nil {
		if err == models.ErrExists {
			utils.ErrorResponse(w, 409)
			return
		}
		if err == models.ErrNotFound {
			utils.ErrorResponse(w, 404)
			return
		}
	}
	utils.Response(w, 200, newU)
}

// -- FORUM --

func (h *Handler) ForumCreate(w http.ResponseWriter, r *http.Request) {
	f := models.Forum{}
	easyjson.UnmarshalFromReader(r.Body, &f)

	forum, err := h.repo.ForumCreate(f)
	if err != nil {
		if err == models.ErrNotFound {
			utils.ErrorResponse(w, 404)
			return
		} else if err == models.ErrExists {
			utils.Response(w, 409, forum)
			return
		}
	}

	utils.Response(w, 201, forum)
}

func (h *Handler) ForumGetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	f, err := h.repo.ForumGetBySlug(slug)
	if  err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, f)
}

func (h *Handler) ForumGetUsers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	since, desc, limit := utils.URLParamsForGetAll(r.URL.Query())

	users, err := h.repo.ForumGetUsers(slug, since, desc, int32(limit))
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, users)
}

// -- THREAD --

func (h *Handler) ThreadCreate(w http.ResponseWriter, r *http.Request) {
	var thread models.Thread

	easyjson.UnmarshalFromReader(r.Body, &thread)
	thread.Forum = chi.URLParam(r, "slug")

	thr, err := h.repo.ThreadCreate(thread)
	if err != nil {
		if err == models.ErrExists {
			utils.Response(w, 409, thr)
			return
		} else if err == models.ErrNotFound {
			utils.ErrorResponse(w, 404)
			return
		}
	}
	
	utils.Response(w, 201, thr)
}


// -- SERVICE 

func (h *Handler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	status := h.repo.GetStatus()
	utils.Response(w, 200, status)
}

func (h *Handler) ServiceClear(w http.ResponseWriter, r *http.Request) {
	h.repo.DropAllData()
}
