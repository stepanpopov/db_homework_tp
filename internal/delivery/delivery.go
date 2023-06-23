package delivery

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, f)
}

func (h *Handler) ForumGetUsers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	since, desc, limit := utils.URLParamsForGetAll(r.URL.Query())

	var sinceUser *string
	if since != "" {
		sinceUser = &since
	}

	users, err := h.repo.ForumGetUsers(slug, sinceUser, desc, int32(limit))
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, users)
}

func (h *Handler) ForumGetThreads(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	since, desc, limit := utils.URLParamsForGetAll(r.URL.Query())

	var sinceTime *time.Time

	sinceTimeParsed, err := time.Parse(time.RFC3339, since)
	if err == nil {
		sinceTime = &sinceTimeParsed
	}

	threads, err := h.repo.ForumGetThreads(slug, sinceTime, desc, int32(limit))
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, threads)
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

func (h *Handler) ThreadGet(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	thr, err := h.repo.ThreadGet(slug)
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, thr)
}

func (h *Handler) ThreadUpdate(w http.ResponseWriter, r *http.Request) {
	slugOrID := chi.URLParam(r, "slug")

	var thread models.Thread
	easyjson.UnmarshalFromReader(r.Body, &thread)
	if id, err := strconv.Atoi(slugOrID); err != nil {
		thread.Slug = slugOrID
	} else {
		thread.ID = int32(id)
	}

	thr, err := h.repo.ThreadUpdate(thread)
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, thr)
}

func (h *Handler) ThreadVote(w http.ResponseWriter, r *http.Request) {
	slugOrID := chi.URLParam(r, "slug")

	var vote models.Vote
	easyjson.UnmarshalFromReader(r.Body, &vote)

	thr, err := h.repo.ThreadVote(slugOrID, vote)
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, thr)
}

// -- POST --
func (h *Handler) PostsCreate(w http.ResponseWriter, r *http.Request) {
	slugOrID := chi.URLParam(r, "slug")

	var posts models.Posts
	easyjson.UnmarshalFromReader(r.Body, &posts)

	newPosts, err := h.repo.PostsCreate(slugOrID, posts)
	if err != nil {
		if err == models.ErrNotFound {
			utils.ErrorResponse(w, 404)
			return
		} else if err == models.ErrRaiseEx {
			utils.ErrorResponse(w, 409)
			return
		}

		utils.ErrorResponse(w, 500)
	}

	utils.Response(w, 201, newPosts)
}

func (h *Handler) PostUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var post models.Post
	easyjson.UnmarshalFromReader(r.Body, &post)
	post.ID, _ = strconv.ParseInt(id, 10, 64)

	newPost, err := h.repo.PostUpdate(post)
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, newPost)
}

func (h *Handler) PostsGetSorted(w http.ResponseWriter, r *http.Request) {
	slugOrID := chi.URLParam(r, "slug")

	URLValues := r.URL.Query()
	sort := URLValues.Get("sort")
	since, desc, limit := utils.URLParamsForGetAll(URLValues)

	var sincePost *int
	if since != "" {
		sincePostParsed, _ := strconv.Atoi(since)
		sincePost = &sincePostParsed
	}

	posts, err := h.repo.PostsGetSorted(slugOrID, sincePost, sort, desc, int32(limit))
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, posts)
}

func (h *Handler) PostGetFull(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idParam)

	relatedQuery := r.URL.Query().Get("related")

	var related []string
	if relatedQuery != "" {
		related = strings.Split(relatedQuery, ",")
	}

	posts, err := h.repo.PostsGetFull(id, related)
	if err != nil {
		utils.ErrorResponse(w, 404)
		return
	}

	utils.Response(w, 200, posts)
}

// -- SERVICE --

func (h *Handler) ServiceStatus(w http.ResponseWriter, r *http.Request) {
	status := h.repo.GetStatus()
	utils.Response(w, 200, status)
}

func (h *Handler) ServiceClear(w http.ResponseWriter, r *http.Request) {
	h.repo.DropAllData()
}
