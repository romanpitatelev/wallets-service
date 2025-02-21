package models

type IP struct {
	Address string `json:"address"`
	Count   int    `json:"count"`
}

type User struct {
	UserID  int  `json:"userid"`
	Deleted bool `json:"deleted"`
}
