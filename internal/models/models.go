package models

import (
	"time"
)

//go:generate easyjson -all .

//easyjson:json
type Error struct {
	Message string `json:"message"`
}

//easyjson:json
type Forum struct {
	ID      int64  `json:"-"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	User    string `json:"user"`
	Posts   int64  `json:"posts,omitempty"`
	Threads int64  `json:"threads,omitempty"`
}

//easyjson:json
type Forums []Forum


//easyjson:json
type User struct {
	Email    string `json:"email"`
	Fullname string `json:"fullname"`
	Nickname string `json:"nickname,omitempty"`
	About    string `json:"about,omitempty"`
}

//easyjson:json
type Users []User

//easyjson:json
type Post struct {
	Author   string    `json:"author"`
	Message  string    `json:"message"`
	Created  time.Time `json:"created,omitempty"`
	Forum    string    `json:"forum,omitempty"`
	ID       int64     `json:"id,omitempty"`
	IsEdited bool      `json:"isEdited,omitempty"`
	Parent   int64     `json:"parent,omitempty"`
	Thread   int32     `json:"thread,omitempty"`
}

//easyjson:json
type Posts []Post

//easyjson:json
type PostUpdate struct {
	Description string `json:"description,omitempty"`
	Message     string `json:"message,omitempty"`
}

//easyjson:json
type PostUpdates []PostUpdate

//easyjson:json
type PostFullInfo struct {
	Author *User   `json:"author,omitempty"`
	Forum  *Forum  `json:"forum,omitempty"`
	Post   *Post   `json:"post,omitempty"`
	Thread *Thread `json:"thread,omitempty"`
}

//easyjson:json
type PostsFulls []PostFullInfo

//easyjson:json
type Thread struct {
	Author  string    `json:"author"`
	Title   string    `json:"title"`
	Message string    `json:"message"`
	ID      int32     `json:"id,omitempty"`
	Forum   string    `json:"forum,omitempty"`
	Created time.Time `json:"created,omitempty"`
	Slug    string    `json:"slug,omitempty"`
	Votes   int32     `json:"votes,omitempty"`
}

//easyjson:json
type Threads []Thread

//easyjson:json
type Vote struct {
	Nickname string `json:"nickname"`
	Voice    int16  `json:"voice"`
}

//easyjson:json
type Votes []Vote

// -- SERVICE

//easyjson:json
type Status struct {
	Forum  int64 `json:"forum"`
	Post   int64 `json:"post"`
	Thread int32 `json:"thread"`
	User   int32 `json:"user"`
}
