package repo

import (
	"context"
	"fmt"

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
	fmt.Println(query)

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

func (repo Repo) ForumGetUsers(slug, sinceUser string, desc bool, limit int32) (models.Users, error) {
	query := ""
	if desc {
		query = fmt.Sprintf(
			`SELECT nickname,
					email,
			   		fullname,
			   		about
			FROM forums_users
			WHERE forum_slug = '%s'
			AND user_nickname < '%s'
			ORDER BY user_nickname DESC
			LIMIT %d`, slug, sinceUser, limit,
		)
	} else {
		query = fmt.Sprintf(
			`SELECT nickname,
					email,
			   		fullname,
			   		about
			FROM forums_users
			WHERE forum_slug = '%s'
			AND user_nickname > '%s'
			ORDER BY user_nickname
			LIMIT %d`, slug, sinceUser, limit,
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

func (repo Repo) ForumGetThreads(slug, sinceUser string, desc bool, limit int32) (models.Threads, error) {
	
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
	fmt.Println(query)

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
					WHERE slug = '%s'`, thread.Slug),
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
