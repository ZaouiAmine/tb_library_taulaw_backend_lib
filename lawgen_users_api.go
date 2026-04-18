package lib

import (
	"encoding/json"
	"io"
	"strconv"

	httpevent "github.com/taubyte/go-sdk/http/event"
)

func handleGetUsersMe(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	uid, ok := bearerUserID(h, db)
	if !ok {
		return writeNestError(h, 401, "unauthorized")
	}
	var u kvUser
	if err := kvGetJSON(db, userKey(uid), &u); err != nil {
		return writeNestError(h, 404, "user not found")
	}
	return writeNest(h, 200, u.publicDTO())
}

func handlePatchUsersMe(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	uid, ok := bearerUserID(h, db)
	if !ok {
		return writeNestError(h, 401, "unauthorized")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	var u kvUser
	if err := kvGetJSON(db, userKey(uid), &u); err != nil {
		return writeNestError(h, 404, "user not found")
	}
	if v, ok := patch["name"].(string); ok {
		u.Name = v
	}
	if v, ok := patch["phone"].(string); ok {
		u.Phone = v
	}
	if v, ok := patch["address"].(string); ok {
		u.Address = v
	}
	if v, ok := patch["description"].(string); ok {
		u.Description = v
	}
	if v, ok := patch["imgUrl"].(string); ok {
		u.ImgURL = v
	}
	if v, ok := patch["imageCover"].(string); ok {
		u.ImageCover = v
	}
	_ = u.save(db)
	return writeNest(h, 200, u.publicDTO())
}

func handlePatchUsersChangePassword(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	uid, ok := bearerUserID(h, db)
	if !ok {
		return writeNestError(h, 401, "unauthorized")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	var u kvUser
	if err := kvGetJSON(db, userKey(uid), &u); err != nil {
		return writeNestError(h, 404, "user not found")
	}
	if u.PasswordHash != hashPassword(in.OldPassword) {
		return writeNestError(h, 401, "invalid password")
	}
	u.PasswordHash = hashPassword(in.NewPassword)
	_ = u.save(db)
	return writeNest(h, 200, map[string]any{"ok": true})
}

func handleGetUsersSearch(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	users, err := listUsers(db)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	users = filterUsersSearch(users, search)
	dtos := make([]map[string]any, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, u.publicDTO())
	}
	return writeNest(h, 200, nestPagination(dtos, page, limit))
}

func handleGetUsersLawyers(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	users, err := listUsers(db)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	users = filterUsersByRole(users, userRoleLawyer)
	users = filterUsersSearch(users, search)
	dtos := make([]map[string]any, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, u.publicDTO())
	}
	return writeNest(h, 200, nestPagination(dtos, page, limit))
}

func handleGetUsersLawyersById(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	id := pathLast(h)
	var u kvUser
	if err := kvGetJSON(db, userKey(id), &u); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, u.publicDTO())
}

func handleGetUsersLawyersLawyers(h httpevent.Event) uint32 {
	return handleGetUsersLawyers(h)
}

func handleGetUsersLawyersGet(h httpevent.Event) uint32 {
	return handleGetUsersLawyers(h)
}

func handleGetUsersLawyersLawyersVerifiying(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	users, err := listUsers(db)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	out := make([]map[string]any, 0)
	for _, u := range users {
		if u.Role == userRoleLawyer && u.VerificationStatus == 0 {
			out = append(out, u.publicDTO())
		}
	}
	return writeNest(h, 200, out)
}

func handleGetUsersLawyersDashboardStats(h httpevent.Event) uint32 {
	return writeNest(h, 200, map[string]any{
		"cases":          0,
		"consultations":  0,
		"applications":   0,
		"notifications": 0,
	})
}

func handleGetUsersJudicialOfficers(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	users, err := listUsers(db)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	out := make([]map[string]any, 0)
	for _, u := range users {
		if u.Role == userRoleOfficer {
			out = append(out, u.publicDTO())
		}
	}
	return writeNest(h, 200, out)
}

func handleGetUsersJudicialOfficersById(h httpevent.Event) uint32 {
	return handleGetUsersLawyersById(h)
}

func handlePutUsersLawyersAcceptByUserId(h httpevent.Event) uint32 {
	return handleLawyerStatus(h, userStatusAccepted)
}

func handlePutUsersLawyersRejectByUserId(h httpevent.Event) uint32 {
	return handleLawyerStatus(h, userStatusPending)
}

func handleLawyerStatus(h httpevent.Event, status int) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	id := pathLast(h)
	var u kvUser
	if err := kvGetJSON(db, userKey(id), &u); err != nil {
		return writeNestError(h, 404, "not found")
	}
	u.Status = status
	_ = u.save(db)
	return writeNest(h, 200, u.publicDTO())
}

func handlePatchUsersLawyersAcceptVerifiyingById(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	id := pathLast(h)
	var u kvUser
	if err := kvGetJSON(db, userKey(id), &u); err != nil {
		return writeNestError(h, 404, "not found")
	}
	u.VerificationStatus = 1
	_ = u.save(db)
	return writeNest(h, 200, u.publicDTO())
}

// --- image / upload stubs (store metadata in KV) ---

func handlePostUsersUploadImage(h httpevent.Event) uint32 {
	return handleUploadPlaceholder(h, "profile")
}

func handlePostUsersUploadImageCover(h httpevent.Event) uint32 {
	return handleUploadPlaceholder(h, "cover")
}

func handlePostUsersCreateImage(h httpevent.Event) uint32 {
	return handleUploadPlaceholder(h, "create")
}

func handlePostUsersSavedAiImage(h httpevent.Event) uint32 {
	return writeNest(h, 201, map[string]any{"url": "/ai/generated.png"})
}

func handlePostUsersRemoveImage(h httpevent.Event) uint32 {
	return handleClearImageField(h, "imgUrl")
}

func handlePostUsersRemoveImageCover(h httpevent.Event) uint32 {
	return handleClearImageField(h, "imageCover")
}

func handleClearImageField(h httpevent.Event, field string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	uid, ok := bearerUserID(h, db)
	if !ok {
		return writeNestError(h, 401, "unauthorized")
	}
	var u kvUser
	if err := kvGetJSON(db, userKey(uid), &u); err != nil {
		return writeNestError(h, 404, "not found")
	}
	if field == "imgUrl" {
		u.ImgURL = ""
	} else {
		u.ImageCover = ""
	}
	_ = u.save(db)
	return writeNest(h, 200, u.publicDTO())
}

func handleUploadPlaceholder(h httpevent.Event, kind string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	uid, ok := bearerUserID(h, db)
	if !ok {
		return writeNestError(h, 401, "unauthorized")
	}
	body, _ := io.ReadAll(h.Body())
	defer h.Body().Close()
	url := "/uploads/" + uid + "/" + kind + "_" + strconv.Itoa(len(body))
	var u kvUser
	_ = kvGetJSON(db, userKey(uid), &u)
	if kind == "cover" {
		u.ImageCover = url
	} else {
		u.ImgURL = url
	}
	_ = u.save(db)
	return writeNest(h, 201, map[string]any{"url": url})
}
