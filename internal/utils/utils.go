package utils

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mailru/easyjson"

	"github.com/stepanpopov/db_homework_tp/internal/models"
)

func PGErr(err error) error {
	if pqError, ok := err.(*pgconn.PgError); ok {
		switch pqError.Code {
		case "23503": // foreign key
			fallthrough
		case "23502": // null
			return models.ErrNotFound
		case "23505": // unique constraint
			return models.ErrExists
		case "P0001":
			return models.ErrRaiseEx
		default:
			return models.ErrInternal
		}
	}
	return nil
}

func Response(w http.ResponseWriter, code int, response easyjson.Marshaler) {
	message, _ := easyjson.Marshal(response)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(message)
}

func ErrorResponse(w http.ResponseWriter, code int) {
	message, _ := easyjson.Marshal(models.Error{Message: "error"})

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(message)
}

func URLParamsForGetAll(values url.Values) (since string, desc bool, limit int) {
	since = values.Get("since")
	descStr := values.Get("desc")
	limitStr := values.Get("limit")

	if descStr == "" || descStr == "false" {
		desc = false
	} else {
		desc = true
	}

	if limitStr == "" {
		limit = 1
	} else {
		limit, _ = strconv.Atoi(limitStr)

	}

	return since, desc, limit
}
