package repo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stepanpopov/db_homework_tp/internal/models"
	"github.com/stepanpopov/db_homework_tp/internal/utils"
)

var ctx = context.Background()

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

// -- USER --
func (r Repo) UserCreate(u models.User) error {
	query := fmt.Sprintf(`INSERT INTO Users
			  (nickname, fullname, about, email) 
			  VALUES ('%s', '%s', '%s', '%s')`, u.Nickname, u.Fullname, u.About, u.Email)

	if _, err := r.db.Exec(ctx, query); err != nil {
		return utils.PGErr(err)
	}

	return nil
}

func (r Repo) UserGetByNicknameOrEmail(nickname, email string) models.Users { // errors ignored
	query := fmt.Sprintf(`SELECT nickname, fullname, about, email 
						  FROM Users
						  WHERE nickname = '%s' OR email = '%s'
						  LIMIT 2`, nickname, email)

	rows, _ := r.db.Query(ctx, query)
	users := make([]models.User, 0, 2)
	for rows.Next() {
		u := models.User{}
		rows.Scan(&u.Nickname, &u.Fullname, &u.About, &u.Email)
		users = append(users, u)
	}
	return users
}

func (r Repo) UserGet(nickname string) (models.User, error) {
	query := fmt.Sprintf(`SELECT nickname, fullname, about, email 
						  FROM Users
						  WHERE nickname = '%s'
						  LIMIT 1`, nickname)

	row := r.db.QueryRow(ctx, query)

	var u models.User
	if err := row.Scan(&u.Nickname, &u.Fullname, &u.About, &u.Email); err != nil {
		return models.User{}, models.ErrNotFound
	}

	return u, nil
}

func (r Repo) UserUpdate(user models.User) (models.User, error) {
	query := fmt.Sprintf(`UPDATE Users
						  SET email    = COALESCE(NULLIF('%s', ''), 	email),
							  fullname = COALESCE(NULLIF('%s', ''), 	fullname),
							  about    = COALESCE(NULLIF('%s', ''), 	about)
						  WHERE nickname = '%s'
						RETURNING nickname, fullname, email, about`,
		user.Email,
		user.Fullname,
		user.About,
		user.Nickname)

	row := r.db.QueryRow(ctx, query)

	var u models.User
	if err := row.Scan(&u.Nickname, &u.Fullname, &u.Email, &u.About); err != nil {
		if err == pgx.ErrNoRows {
			return models.User{}, models.ErrNotFound
		} else {
			return models.User{}, utils.PGErr(err)
		}

	}

	return u, nil
}

// -- FORUM --
func (r Repo) ForumCreate(forum models.Forum) (models.Forum, error) {
	con, _ := r.db.Acquire(ctx)

	query := fmt.Sprintf(
		`	INSERT INTO forums (slug, title, owner_nickname)
				VALUES ('%s', '%s', (
					SELECT nickname FROM users WHERE nickname = '%s'
		)) RETURNING owner_nickname`,
		forum.Slug,
		forum.Title,
		forum.User,
	)

	defer con.Release()

	if err := con.QueryRow(ctx, query).Scan(&forum.User); err != nil {
		pgErr := utils.PGErr(err)
		if pgErr == models.ErrExists {
			newForum := models.Forum{}

			con.QueryRow(context.Background(), fmt.Sprintf(
				`SELECT slug, title, threads, posts, owner_nickname
					FROM forums
					WHERE slug = '%s'`,
				forum.Slug),
			).Scan(
				&newForum.Slug,
				&newForum.Title,
				&newForum.Threads,
				&newForum.Posts,
				&newForum.User,
			)
			return newForum, pgErr
		} else {
			return models.Forum{}, pgErr
		}
	}

	return forum, nil
}

func (repo Repo) ForumGetBySlug(slug string) (models.Forum, error) {
	query := fmt.Sprintf(
		`SELECT slug, title, threads, posts, owner_nickname
			FROM forums
			WHERE slug = '%s'`, slug)

	var forum models.Forum
	if err := repo.db.QueryRow(context.Background(), query).Scan(
		&forum.Slug,
		&forum.Title,
		&forum.Threads,
		&forum.Posts,
		&forum.User,
	); err != nil {
		return models.Forum{}, models.ErrNotFound
	}

	return forum, nil
}

func (repo Repo) ForumGetUsers(slug string, sinceUser *string, desc bool, limit int32) (models.Users, error) {
	orderParam := "DESC"
	cmpParam := "<"
	if !desc {
		orderParam = "ASC"
		cmpParam = ">"
	}

	query := ""
	if sinceUser != nil {
		query = fmt.Sprintf(
			`SELECT nickname,
					email,
			   		fullname,
			   		about
			FROM forums_users
			WHERE forum_slug = '%s'
			AND user_nickname %s '%s'
			ORDER BY user_nickname %s
			LIMIT %d`, slug, cmpParam, *sinceUser, orderParam, limit,
		)
	} else {
		query = fmt.Sprintf(
			`SELECT nickname,
					email,
			   		fullname,
			   		about
			FROM forums_users
			WHERE forum_slug = '%s'
			ORDER BY user_nickname %s
			LIMIT %d`, slug, orderParam, limit,
		)
	}

	rows, _ := repo.db.Query(ctx, query)
	defer rows.Close()

	if len(rows.RawValues()) == 0 {
		var forumExists bool
		repo.db.QueryRow(ctx, fmt.Sprintf(
			`SELECT EXISTS(
				SELECT 1
				FROM forums
				WHERE slug = '%s'
			 	LIMIT 1)`, slug),
		).Scan(
			&forumExists,
		)

		if !forumExists {
			return nil, models.ErrNotFound
		}
	}

	users := make(models.Users, 0)
	for rows.Next() {
		var user models.User
		rows.Scan(
			&user.Nickname,
			&user.Email,
			&user.Fullname,
			&user.About,
		)

		users = append(users, user)
	}

	return users, nil
}

func (r Repo) ForumGetThreads(forumSlug string, sinceTime *time.Time, desc bool, limit int32) (models.Threads, error) {
	orderParam := "DESC"
	cmpParam := "<="
	if !desc {
		orderParam = "ASC"
		cmpParam = ">="
	}

	query := ""
	var rows pgx.Rows
	if sinceTime != nil {
		query = fmt.Sprintf(`
		SELECT id,
			slug,
			forum_slug,
			author_nickname,
			title,
			message,
			votes,
			created
		FROM threads
		WHERE forum_slug = '%s' AND created %s $1
		ORDER BY created %s
		LIMIT %d`, forumSlug, cmpParam, orderParam, limit)

		rows, _ = r.db.Query(ctx, query, sinceTime)
	} else {
		query = fmt.Sprintf(`
		SELECT id,
			slug,
			forum_slug,
			author_nickname,
			title,
			message,
			votes,
			created
		FROM threads
		WHERE forum_slug = '%s'
		ORDER BY created %s
		LIMIT %d`, forumSlug, orderParam, limit)

		rows, _ = r.db.Query(ctx, query)
	}

	defer rows.Close()

	if len(rows.RawValues()) == 0 {
		var forumExists bool
		r.db.QueryRow(ctx, fmt.Sprintf(
			`SELECT EXISTS(
				SELECT 1
				FROM forums
				WHERE slug = '%s'
			 	LIMIT 1)`, forumSlug),
		).Scan(
			&forumExists,
		)

		if !forumExists {
			return nil, models.ErrNotFound
		}
	}

	threads := make(models.Threads, 0)
	for rows.Next() {
		var thread models.Thread
		var threadSLugNullable *string
		rows.Scan(
			&thread.ID,
			&threadSLugNullable,
			&thread.Forum,
			&thread.Author,
			&thread.Title,
			&thread.Message,
			&thread.Votes,
			&thread.Created,
		)

		if threadSLugNullable != nil {
			thread.Slug = *threadSLugNullable
		}

		threads = append(threads, thread)
	}

	return threads, nil
}

// -- THREAD --

func (r Repo) ThreadCreate(thread models.Thread) (models.Thread, error) {
	con, _ := r.db.Acquire(ctx)
	defer con.Release()

	// thrCreatedRFC339 := thread.Created.Format("2023-05-02T09:34:01Z")

	query := ""
	if thread.Slug == "" {
		query = fmt.Sprintf(`
					INSERT INTO threads (author_nickname, title, message, forum_slug, created)
					VALUES ((SELECT nickname FROM users WHERE nickname = '%s'), '%s', '%s',
							(SELECT slug FROM forums WHERE slug = '%s'), $1)
					RETURNING id, author_nickname, forum_slug, created`,
			thread.Author,
			thread.Title,
			thread.Message,
			thread.Forum,
			// thrCreatedRFC339,
		)
	} else {
		query = fmt.Sprintf(`
					INSERT INTO threads (slug, author_nickname, title, message, forum_slug, created)
					VALUES ('%s', (SELECT nickname FROM users WHERE nickname = '%s'), '%s', '%s',
							(SELECT slug FROM forums WHERE slug = '%s'), $1)
					RETURNING id, author_nickname, forum_slug, created`,
			thread.Slug,
			thread.Author,
			thread.Title,
			thread.Message,
			thread.Forum,
			// thrCreatedRFC339,
		)
	}

	if err := con.QueryRow(ctx, query, thread.Created).Scan(
		&thread.ID,
		&thread.Author,
		&thread.Forum,
		&thread.Created,
	); err != nil {
		pgErr := utils.PGErr(err)
		if pgErr == models.ErrExists {
			
			con.QueryRow(ctx, fmt.Sprintf(`
					SELECT id,
						slug,
						forum_slug,
						author_nickname,
						title,
						message,
						votes,
						created
					FROM threads
					WHERE slug = '%s'
					LIMIT 1`, thread.Slug),
			).Scan(
				&thread.ID,
				&thread.Slug,
				&thread.Forum,
				&thread.Author,
				&thread.Title,
				&thread.Message,
				&thread.Votes,
				&thread.Created,
			)
			return thread, pgErr
		} else {
			return models.Thread{}, pgErr
		}
	}

	return thread, nil
}

func (r Repo) ThreadGet(slugOrID string) (models.Thread, error) {
	var thread models.Thread

	whereStatement := ""
	if _, err := strconv.Atoi(slugOrID); err != nil {
		whereStatement = fmt.Sprintf("slug = '%s'", slugOrID)
	} else {
		whereStatement = fmt.Sprintf("id = %s", slugOrID)
	}

	query := fmt.Sprintf(`
		SELECT 
			id,
			slug,
			forum_slug,
			author_nickname,
			title,
			message,
			votes,
			created
		FROM threads
		WHERE %s
		LIMIT 1`,
		whereStatement,
	)

	var threadSLugNullable *string
	if err := r.db.QueryRow(ctx, query).Scan(
		&thread.ID,
		&threadSLugNullable,
		&thread.Forum,
		&thread.Author,
		&thread.Title,
		&thread.Message,
		&thread.Votes,
		&thread.Created,
	); err != nil {
		return models.Thread{}, models.ErrNotFound
	}

	if threadSLugNullable != nil {
		thread.Slug = *threadSLugNullable
	}

	return thread, nil
}

func (r Repo) ThreadUpdate(thr models.Thread) (models.Thread, error) {
	whereStatement := ""
	if thr.ID == 0 {
		whereStatement = fmt.Sprintf("slug = '%s'", thr.Slug)
	} else {
		whereStatement = fmt.Sprintf("id = %d", thr.ID)
	}

	var threadSLugNullable *string
	var thread models.Thread
	if err := r.db.QueryRow(ctx, fmt.Sprintf(`
			UPDATE threads
			SET title   = COALESCE(NULLIF('%s', ''), title),
				message = COALESCE(NULLIF('%s', ''), message)
			WHERE %s
			RETURNING id, slug, forum_slug, author_nickname, title, message, votes, created`,
		thr.Title,
		thr.Message,
		whereStatement),
	).Scan(
		&thread.ID,
		&threadSLugNullable,
		&thread.Forum,
		&thread.Author,
		&thread.Title,
		&thread.Message,
		&thread.Votes,
		&thread.Created,
	); err != nil {
		return models.Thread{}, err
	}

	if threadSLugNullable != nil {
		thread.Slug = *threadSLugNullable
	}

	return thread, nil
}

func (r Repo) ThreadVote(slugOrID string, vote models.Vote) (models.Thread, error) {
	tx, _ := r.db.Begin(ctx)

	thread := models.Thread{}

	whereStatement := ""
	if _, err := strconv.Atoi(slugOrID); err != nil {
		whereStatement = fmt.Sprintf("slug = '%s'", slugOrID)
	} else {
		whereStatement = fmt.Sprintf("id = %s", slugOrID)
	}

	query := fmt.Sprintf(`
		SELECT 
			id,
			slug,
			forum_slug,
			author_nickname,
			title,
			message,
			created
		FROM threads
		WHERE %s
		LIMIT 1`,
		whereStatement,
	)

	var threadSLugNullable *string
	if err := tx.QueryRow(ctx, query).Scan(
		&thread.ID,
		&threadSLugNullable,
		&thread.Forum,
		&thread.Author,
		&thread.Title,
		&thread.Message,
		&thread.Created,
	); err != nil {
		tx.Commit(ctx)
		return models.Thread{}, models.ErrNotFound
	}

	if threadSLugNullable != nil {
		thread.Slug = *threadSLugNullable
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO votes (thread_id, author_nickname, voice)
			VALUES (%d, '%s', %d)
			ON CONFLICT (thread_id, author_nickname) DO UPDATE SET voice = %d`,
		thread.ID,
		vote.Nickname,
		vote.Voice,
		vote.Voice),
	); err != nil {
		tx.Rollback(ctx) // TEST
		return models.Thread{}, models.ErrNotFound
	}

	tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT votes FROM threads WHERE id = %d`, thread.ID), // update votes in thread
	).Scan(
		&thread.Votes,
	)

	tx.Commit(ctx)
	return thread, nil
}

// -- POST --

func (r Repo) PostsCreate(slugOrID string, posts models.Posts) (models.Posts, error) {
	tx, _ := r.db.Begin(ctx)

	whereStatement := ""
	if _, err := strconv.Atoi(slugOrID); err != nil {
		whereStatement = fmt.Sprintf("slug = '%s'", slugOrID)
	} else {
		whereStatement = fmt.Sprintf("id = %s", slugOrID)
	}

	queryThread := fmt.Sprintf(`
			SELECT id, forum_slug
			FROM threads
			WHERE %s
			LIMIT 1`,
		whereStatement,
	)

	var threadID int
	var forumSlug string
	if err := tx.QueryRow(ctx, queryThread).Scan(&threadID, &forumSlug); err != nil {
		tx.Commit(ctx)
		return nil, models.ErrNotFound
	}

	if len(posts) == 0 {
		tx.Commit(ctx)
		return posts, nil
	}

	// BATCH
	batch := new(pgx.Batch)

	for _, postModel := range posts {
		parentStr := "NULL"
		if postModel.Parent != 0 {
			parentStr = fmt.Sprintf("%d", postModel.Parent)
		}

		batch.Queue(fmt.Sprintf(`
				INSERT INTO posts (thread_id, author_nickname, forum_slug, message, parent)
				VALUES (%d, '%s', '%s', '%s', %s) 
				RETURNING id, thread_id, author_nickname, forum_slug, is_edited, message, parent, created`,
			threadID,
			postModel.Author,
			forumSlug,
			postModel.Message,
			parentStr),
		)
	}

	insertedPosts := make(models.Posts, batch.Len())
	batchResults := tx.SendBatch(ctx, batch)

	for i := 0; i < batch.Len(); i++ {
		var postParent *int64

		if err := batchResults.QueryRow().Scan(
			&insertedPosts[i].ID,
			&insertedPosts[i].Thread,
			&insertedPosts[i].Author,
			&insertedPosts[i].Forum,
			&insertedPosts[i].IsEdited,
			&insertedPosts[i].Message,
			&postParent,
			&insertedPosts[i].Created,
		); err != nil {
			batchResults.Close()
			tx.Rollback(ctx) // TEST

			return nil, utils.PGErr(err)
		}

		if postParent != nil {
			insertedPosts[i].Parent = *postParent
		}
	}

	batchResults.Close()
	tx.Commit(ctx)
	return insertedPosts, nil
}

func (r Repo) PostUpdate(post models.Post) (models.Post, error) {
	var postMessageStr = "NULL"
	if post.Message != "" {
		postMessageStr = fmt.Sprintf("'%s'", post.Message)
	}

	query := fmt.Sprintf(`
		UPDATE posts
		SET message   = COALESCE(%s, message),
			is_edited = CASE
							WHEN (is_edited = TRUE
								OR (is_edited = FALSE AND %s IS NOT NULL AND %s <> message)) THEN TRUE
							ELSE FALSE
				END
		WHERE id = %d
		RETURNING id, thread_id, author_nickname, forum_slug, is_edited, message, parent, created`,
		postMessageStr,
		postMessageStr,
		postMessageStr,
		post.ID,
	)

	var postParent *int64

	if err := r.db.QueryRow(ctx, query).Scan(
		&post.ID,
		&post.Thread,
		&post.Author,
		&post.Forum,
		&post.IsEdited,
		&post.Message,
		&postParent,
		&post.Created,
	); err != nil {
		return models.Post{}, err
	}

	if postParent != nil {
		post.Parent = *postParent
	}

	return post, nil
}

func (r Repo) PostsGetSorted(threadSlugOrId string, sincePost *int, sort string, desc bool, limit int32) (models.Posts, error) {
	if sort == "" {
		sort = "flat"
	}

	whereStatement := ""
	if _, err := strconv.Atoi(threadSlugOrId); err != nil {
		whereStatement = fmt.Sprintf("slug = '%s'", threadSlugOrId)
	} else {
		whereStatement = fmt.Sprintf("id = %s", threadSlugOrId)
	}

	queryThread := fmt.Sprintf(`
			SELECT id
			FROM threads
			WHERE %s
			LIMIT 1`,
		whereStatement,
	)

	con, _ := r.db.Acquire(ctx)
	defer con.Release()

	var threadID int
	if err := con.QueryRow(ctx, queryThread).Scan(&threadID); err != nil {
		return nil, models.ErrNotFound
	}

	orderParam := "DESC"
	cmpParam := "<"
	orderParentTreeParam := "ORDER BY path[1] DESC, path[2:]"
	if !desc {
		orderParam = "ASC"
		cmpParam = ">"
		orderParentTreeParam = "ORDER BY path"
	}

	query := ""

	if sincePost != nil {
		switch sort {
		case "flat":
			query = fmt.Sprintf(`
				SELECT id,
					thread_id,
					author_nickname,
					forum_slug,
					is_edited,
					message,
					parent,
					created
				FROM posts
				WHERE thread_id = %d
				AND id %s %d
				ORDER BY id %s
				LIMIT %d`, threadID, cmpParam, *sincePost, orderParam, limit,
			)
		case "tree":
			query = fmt.Sprintf(`
				SELECT id,
					thread_id,
					author_nickname,
					forum_slug,
					is_edited,
					message,
					parent,
					created
				FROM posts
				WHERE thread_id = %d
				AND path %s (SELECT path FROM posts WHERE id = %d)
				ORDER BY path %s
				LIMIT %d`, threadID, cmpParam, *sincePost, orderParam, limit,
			)
		case "parent_tree":
			query = fmt.Sprintf(`
				WITH roots AS (
					SELECT DISTINCT path[1]
					FROM posts
					WHERE thread_id = %d
					AND parent IS NULL
					AND path[1] %s (SELECT path[1] FROM posts WHERE id = %d)
					ORDER BY path[1] %s
					LIMIT %d
				)
				SELECT id,
					thread_id,
					author_nickname,
					forum_slug,
					is_edited,
					message,
					parent,
					created
				FROM posts
				WHERE thread_id = %d
				AND path[1] IN (SELECT * FROM roots)
				%s`, threadID, cmpParam, *sincePost, orderParam, limit, threadID, orderParentTreeParam,
			)
		}
	} else {
		switch sort {
		case "flat":
			query = fmt.Sprintf(`
				SELECT id,
					thread_id,
					author_nickname,
					forum_slug,
					is_edited,
					message,
					parent,
					created
				FROM posts
				WHERE thread_id = %d
				ORDER BY id %s
				LIMIT %d`, threadID, orderParam, limit,
			)
		case "tree":
			query = fmt.Sprintf(`
				SELECT id,
					thread_id,
					author_nickname,
					forum_slug,
					is_edited,
					message,
					parent,
					created
				FROM posts
				WHERE thread_id = %d
				ORDER BY path %s
				LIMIT %d`, threadID, orderParam, limit,
			)
		case "parent_tree":
			query = fmt.Sprintf(`
				WITH roots AS (
					SELECT DISTINCT path[1]
					FROM posts
					WHERE thread_id = %d
					ORDER BY path[1] %s
					LIMIT %d
				)
				SELECT id,
					thread_id,
					author_nickname,
					forum_slug,
					is_edited,
					message,
					parent,
					created
				FROM posts
				WHERE thread_id = %d
				AND path[1] IN (SELECT * FROM roots)
				%s`, threadID, orderParam, limit, threadID, orderParentTreeParam,
			)
		}
	}

	rows, _ := con.Query(ctx, query)
	defer rows.Close()

	posts := make(models.Posts, 0)
	for rows.Next() {
		var post models.Post
		var postParent *int64
		rows.Scan(
			&post.ID,
			&post.Thread,
			&post.Author,
			&post.Forum,
			&post.IsEdited,
			&post.Message,
			&postParent,
			&post.Created,
		)

		if postParent != nil {
			post.Parent = *postParent
		}

		posts = append(posts, post)
	}
	return posts, nil
}

func (r Repo) PostsGetFull(id int, related []string) (models.PostFullInfo, error) {
	con, _ := r.db.Acquire(ctx)
	defer con.Release()

	var post models.Post
	var postParent *int64

	if err := con.QueryRow(ctx, fmt.Sprintf(`
			SELECT id,
				   thread_id,
				   author_nickname,
				   forum_slug,
				   is_edited,
				   message,
				   parent,
				   created
			FROM posts
			WHERE id = %d
			LIMIT 1`, id),
	).Scan(
		&post.ID,
		&post.Thread,
		&post.Author,
		&post.Forum,
		&post.IsEdited,
		&post.Message,
		&postParent,
		&post.Created,
	); err != nil {
		return models.PostFullInfo{}, models.ErrNotFound
	}

	if postParent != nil {
		post.Parent = *postParent
	}

	postFull := models.PostFullInfo{
		Post: &post,
	}

	for _, relation := range related {
		switch relation {
		case "forum":
			query := fmt.Sprintf(
				`SELECT slug, title, threads, posts, owner_nickname
					FROM forums
					WHERE slug = '%s'
					LIMIT 1`, post.Forum)

			var forum models.Forum
			con.QueryRow(context.Background(), query).Scan(
				&forum.Slug,
				&forum.Title,
				&forum.Threads,
				&forum.Posts,
				&forum.User,
			)
			postFull.Forum = &forum

		case "thread":
			query := fmt.Sprintf(`
				SELECT 
					id,
					slug,
					forum_slug,
					author_nickname,
					title,
					message,
					votes,
					created
				FROM threads
				WHERE id = %d
				LIMIT 1`,
				post.Thread,
			)
			var threadSLugNullable *string
			var thread models.Thread
			con.QueryRow(ctx, query).Scan(
				&thread.ID,
				&threadSLugNullable,
				&thread.Forum,
				&thread.Author,
				&thread.Title,
				&thread.Message,
				&thread.Votes,
				&thread.Created,
			)
			if threadSLugNullable != nil {
				thread.Slug = *threadSLugNullable
			}
			postFull.Thread = &thread

		case "user":
			query := fmt.Sprintf(`SELECT nickname, fullname, about, email 
						  FROM Users
						  WHERE nickname = '%s'
						  LIMIT 1`, post.Author)
			var u models.User
			con.QueryRow(ctx, query).Scan(&u.Nickname, &u.Fullname, &u.About, &u.Email)
			postFull.Author = &u
		}

	}

	return postFull, nil
}

// -- SERVICE --
func (repo Repo) DropAllData() error {
	_, err := repo.db.Exec(ctx,
		`TRUNCATE users, forums, threads, votes, posts, forums_users_nicknames`)
	return err
}

func (repo Repo) GetStatus() (status models.Status) {
	ctx := context.Background()

	tx, _ := repo.db.Begin(ctx)

	tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&status.User)
	tx.QueryRow(ctx, `SELECT COUNT(*) FROM forums`).Scan(&status.Forum)
	tx.QueryRow(ctx, `SELECT COUNT(*) FROM threads`).Scan(&status.Thread)
	tx.QueryRow(ctx, `SELECT COUNT(*) FROM posts`).Scan(&status.Post)

	tx.Commit(ctx)

	return status
}
