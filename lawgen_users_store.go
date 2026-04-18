package lib

import (
	"strings"

	"github.com/taubyte/go-sdk/database"
)

const (
	userRoleAdmin    = 1
	userRoleLawyer   = 2
	userRoleOfficer  = 3
	userRoleClient   = 4
	userStatusPending   = 0
	userStatusAccepted  = 1
)

func userKey(id string) string {
	return "users/" + id
}

func userEmailIndexKey(email string) string {
	return "users_email/" + normalize(email)
}

func (u *kvUser) save(db database.Database) error {
	if err := kvPutJSON(db, userKey(u.ID), u); err != nil {
		return err
	}
	return kvPutJSON(db, userEmailIndexKey(u.Email), map[string]string{"id": u.ID})
}

func findUserByEmail(db database.Database, email string) (*kvUser, error) {
	var ref struct {
		ID string `json:"id"`
	}
	if err := kvGetJSON(db, userEmailIndexKey(email), &ref); err != nil {
		return nil, err
	}
	var u kvUser
	if err := kvGetJSON(db, userKey(ref.ID), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func listUsers(db database.Database) ([]kvUser, error) {
	keys, err := kvListKeys(db, "users/")
	if err != nil {
		return nil, err
	}
	out := make([]kvUser, 0, len(keys))
	for _, k := range keys {
		var u kvUser
		if err := kvGetJSON(db, k, &u); err != nil {
			continue
		}
		if u.ID != "" {
			out = append(out, u)
		}
	}
	return out, nil
}

func filterUsersByRole(users []kvUser, role int) []kvUser {
	out := make([]kvUser, 0)
	for _, u := range users {
		if u.Role == role {
			out = append(out, u)
		}
	}
	return out
}

func filterUsersSearch(users []kvUser, term string) []kvUser {
	if term == "" {
		return users
	}
	out := make([]kvUser, 0)
	for _, u := range users {
		if strings.Contains(normalize(u.Name), term) || strings.Contains(normalize(u.Email), term) {
			out = append(out, u)
		}
	}
	return out
}
