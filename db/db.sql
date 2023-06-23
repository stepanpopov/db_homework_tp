DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;

CREATE EXTENSION IF NOT EXISTS CITEXT;

CREATE UNLOGGED TABLE IF NOT EXISTS users
(
    nickname CITEXT COLLATE ucs_basic                   PRIMARY KEY,
    email    CITEXT UNIQUE                               NOT NULL,
    fullname TEXT                                        NOT NULL,
    about    TEXT                                        NOT NULL
);

CREATE UNLOGGED TABLE IF NOT EXISTS forums
(
    slug           CITEXT                    PRIMARY KEY,
    title          TEXT                      NOT NULL,
    threads        INTEGER                   NOT NULL DEFAULT 0,
    posts          INTEGER                   NOT NULL DEFAULT 0,
    owner_nickname CITEXT                    NOT NULL,

    FOREIGN KEY (owner_nickname) REFERENCES users (nickname) 
);

CREATE UNLOGGED TABLE IF NOT EXISTS forums_users_nicknames
(
    forum_slug    CITEXT                   NOT NULL,
    user_nickname CITEXT COLLATE ucs_basic NOT NULL,

    PRIMARY KEY (forum_slug, user_nickname)
);

CREATE VIEW forums_users AS
SELECT fu_nicknames.forum_slug, fu_nicknames.user_nickname, u.nickname, u.email, u.fullname, u.about
FROM forums_users_nicknames AS fu_nicknames
         JOIN users AS u ON fu_nicknames.user_nickname = u.nickname;

CREATE UNLOGGED TABLE IF NOT EXISTS threads
(
    id              SERIAL PRIMARY KEY,
    slug            CITEXT UNIQUE,
    forum_slug      CITEXT                                                NOT NULL,
    author_nickname CITEXT                                                NOT NULL,
    title           TEXT                                                  NOT NULL,
    message         TEXT                                                  NOT NULL,
    votes           INTEGER                     DEFAULT 0                 NOT NULL,
    created         TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE UNLOGGED TABLE IF NOT EXISTS votes
(
    thread_id       INTEGER  NOT NULL,
    author_nickname CITEXT   NOT NULL,
    voice           SMALLINT NOT NULL,

    FOREIGN KEY (author_nickname) REFERENCES users (nickname),

    PRIMARY KEY (thread_id, author_nickname)
);

CREATE UNLOGGED TABLE IF NOT EXISTS posts
(
    id              BIGSERIAL PRIMARY KEY,
    thread_id       INT                                                   NOT NULL,
    author_nickname CITEXT                                                NOT NULL,
    forum_slug      CITEXT                                                NOT NULL,
    is_edited       BOOLEAN                     DEFAULT FALSE             NOT NULL,
    message         TEXT                                                  NOT NULL,
    parent          INT                                                ,  -- mmmm
    created         TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    path            INTEGER[]                                              NOT NULL,

    FOREIGN KEY (author_nickname) REFERENCES users (nickname)  -- BINGO
);

-- Triggers and procedures

DROP FUNCTION IF EXISTS add_new_forum_user() CASCADE;
CREATE OR REPLACE FUNCTION add_new_forum_user() RETURNS TRIGGER AS
$add_forum_user$
BEGIN
    INSERT INTO forums_users_nicknames (forum_slug, user_nickname)
    VALUES (NEW.forum_slug, NEW.author_nickname)
    ON CONFLICT DO NOTHING;

    RETURN NULL;
END;
$add_forum_user$ LANGUAGE plpgsql;

CREATE TRIGGER add_new_forum_user_after_insert_in_threads
    AFTER INSERT
    ON threads
    FOR EACH ROW
EXECUTE PROCEDURE add_new_forum_user();

CREATE TRIGGER add_new_forum_user_after_insert_in_posts
    AFTER INSERT
    ON posts
    FOR EACH ROW
EXECUTE PROCEDURE add_new_forum_user();

--

DROP FUNCTION IF EXISTS increment_forum_threads() CASCADE;
CREATE OR REPLACE FUNCTION increment_forum_threads() RETURNS TRIGGER AS
$increment_forum_threads$
BEGIN
    UPDATE forums
    SET threads = threads + 1
    WHERE slug = NEW.forum_slug;

    RETURN NULL;
END;
$increment_forum_threads$ LANGUAGE plpgsql;

CREATE TRIGGER increment_forum_threads_after_insert_on_threads
    AFTER INSERT
    ON threads
    FOR EACH ROW
EXECUTE PROCEDURE increment_forum_threads();

--

DROP FUNCTION IF EXISTS insert_vote() CASCADE;
CREATE OR REPLACE FUNCTION insert_vote() RETURNS TRIGGER AS
$insert_vote$
BEGIN
    UPDATE threads
    SET votes = votes + NEW.voice
    WHERE id = NEW.thread_id;

    RETURN NULL;
END;
$insert_vote$ LANGUAGE plpgsql;

CREATE TRIGGER insert_vote_after_insert_on_threads
    AFTER INSERT
    ON votes
    FOR EACH ROW
EXECUTE PROCEDURE insert_vote();

--

DROP FUNCTION IF EXISTS update_vote() CASCADE;
CREATE OR REPLACE FUNCTION update_vote() RETURNS TRIGGER AS
$update_vote$
BEGIN
    UPDATE threads
    SET votes = votes + NEW.voice - OLD.voice
    WHERE id = NEW.thread_id;

    RETURN NULL;
END;
$update_vote$ LANGUAGE plpgsql;

CREATE TRIGGER update_vote_after_insert_on_threads
    AFTER UPDATE
    ON votes
    FOR EACH ROW
EXECUTE PROCEDURE update_vote();

--

DROP FUNCTION IF EXISTS increment_forum_posts() CASCADE;
CREATE OR REPLACE FUNCTION increment_forum_posts() RETURNS TRIGGER AS
$increment_forum_posts$
BEGIN
    UPDATE forums
    SET posts = posts + 1
    WHERE slug = NEW.forum_slug;

    RETURN NEW;
END;
$increment_forum_posts$ LANGUAGE plpgsql;

CREATE TRIGGER increment_forum_posts_after_insert_on_threads
    BEFORE INSERT
    ON posts
    FOR EACH ROW
EXECUTE PROCEDURE increment_forum_posts();

--

CREATE OR REPLACE FUNCTION add_path_to_post() RETURNS TRIGGER AS
$add_path_to_post$
DECLARE
    parent_path INTEGER[];
BEGIN
    IF NEW.parent IS NULL THEN
        NEW.path := NEW.path || NEW.id;
    ELSE
        SELECT path
        FROM posts
        WHERE id = NEW.parent
          AND thread_id = NEW.thread_id
        INTO parent_path;

        IF (COALESCE(ARRAY_LENGTH(parent_path, 1), 0) = 0) THEN
            RAISE EXCEPTION
                'parent post not exists';
        END IF;

        NEW.path := NEW.path || parent_path || NEW.id;
    END IF;
    RETURN NEW;
END;
$add_path_to_post$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS add_path_to_post ON posts CASCADE;

CREATE TRIGGER add_path_to_post
    BEFORE INSERT
    ON posts
    FOR EACH ROW
EXECUTE PROCEDURE add_path_to_post();


CREATE INDEX IF NOT EXISTS users_nickname_index ON users USING hash (nickname);
CREATE INDEX IF NOT EXISTS users_email_index ON users USING hash (email);
CREATE INDEX IF NOT EXISTS forum_slug_index ON forums USING hash (slug);
CREATE INDEX IF NOT EXISTS thread_slug_index ON threads USING hash (slug);
CREATE INDEX IF NOT EXISTS thread_id_index ON threads USING hash (id);
CREATE INDEX IF NOT EXISTS post_id_index ON posts USING hash (id);

--
CREATE INDEX IF NOT EXISTS posts_thread_id_path1_id_idx ON posts (thread_id, (path[1]), id);
CREATE INDEX IF NOT EXISTS posts_thread_id_path_idx ON posts (thread_id, path, id);
CREATE INDEX IF NOT EXISTS posts_thread_id_parent_path_idx ON posts (thread_id, parent, path);
CREATE INDEX IF NOT EXISTS posts_parent_id_idx ON posts (parent, id);
CREATE INDEX IF NOT EXISTS posts_id_created_thread_id_idx ON posts (id, created, thread_id);
CREATE INDEX IF NOT EXISTS posts_id_path_idx ON posts (id, path);
--

CREATE INDEX IF NOT EXISTS threads_forum_slug_created_idx ON threads (forum_slug, created);

ANALYZE;
VACUUM FULL;
VACUUM ANALYZE;
