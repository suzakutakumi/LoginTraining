package main

type InputUserInfo struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

type UserInfo struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

type DBUser struct {
	Id       string `db:"id"`
	Password string `db:"password"`
	Name     string `db:"name"`
}

type Activate struct {
	Id   string `db:"id"`
	UUID string `db:"uuid"`
}

type Token struct {
	Id   string `db:"id"`
	UUID string `db:"uuid"`
}
