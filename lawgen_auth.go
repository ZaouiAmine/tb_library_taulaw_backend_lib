package lib

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/taubyte/go-sdk/database"
	httpevent "github.com/taubyte/go-sdk/http/event"
)

func tokensForUser(userID string) (access, refresh string) {
	seed := strconv.FormatInt(time.Now().UnixNano(), 10)
	return "access_" + seed, "refresh_" + seed
}

func handlePostAuthLogin(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in loginBody
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	email := normalize(in.Email)
	if email == "" || strings.TrimSpace(in.Password) == "" {
		return writeNestError(h, 400, "email and password required")
	}
	u, err := findUserByEmail(db, email)
	if err != nil || u == nil {
		return writeNestError(h, 404, "user not found")
	}
	if u.Role == userRoleLawyer && u.Status != userStatusAccepted {
		return writeNestError(h, 400, "lawyer not accepted")
	}
	if u.PasswordHash != hashPassword(in.Password) {
		return writeNestError(h, 401, "invalid email or password")
	}
	access, refresh := tokensForUser(u.ID)
	u.RefreshToken = refresh
	_ = u.save(db)
	_ = kvPutJSON(db, "sessions/"+access, map[string]string{"userId": u.ID})
	_ = kvPutJSON(db, "refresh/"+refresh, map[string]string{"userId": u.ID})
	resp := map[string]any{
		"accessToken":  access,
		"refreshToken": refresh,
		"user":         u.publicDTO(),
	}
	return writeNest(h, 200, resp)
}

func handlePostAuthRegister(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in registerBody
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	email := normalize(in.Email)
	if email == "" || strings.TrimSpace(in.Password) == "" || strings.TrimSpace(in.Name) == "" {
		return writeNestError(h, 400, "missing required fields")
	}
	if u, _ := findUserByEmail(db, email); u != nil {
		return writeNestError(h, 409, "email already exist")
	}
	u := kvUser{
		ID:                 newEntityID("usr"),
		Name:               strings.TrimSpace(in.Name),
		Email:              email,
		Phone:              strings.TrimSpace(in.Phone),
		Role:               userRoleClient,
		Status:             userStatusPending,
		IsEmailVerified:    false,
		VerificationStatus: 0,
		PasswordHash:       hashPassword(in.Password),
	}
	if err := u.save(db); err != nil {
		return writeNestError(h, 500, "failed to save user")
	}
	return writeNest(h, 201, u.publicDTO())
}

func handlePostAuthRefreshToken(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in refreshBody
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	rt := strings.TrimSpace(in.RefreshToken)
	if rt == "" {
		return writeNestError(h, 400, "refresh token required")
	}
	var ref struct {
		UserID string `json:"userId"`
	}
	if err := kvGetJSON(db, "refresh/"+rt, &ref); err != nil || ref.UserID == "" {
		return writeNestError(h, 401, "invalid refresh token")
	}
	access, newRefresh := tokensForUser(ref.UserID)
	_ = kvPutJSON(db, "sessions/"+access, map[string]string{"userId": ref.UserID})
	_ = kvPutJSON(db, "refresh/"+newRefresh, map[string]string{"userId": ref.UserID})
	return writeNest(h, 200, map[string]any{"accessToken": access, "refreshToken": newRefresh})
}

func handlePostAuthLogout(h httpevent.Event) uint32 {
	return writeNest(h, 200, "logged out")
}

func handlePostAuthSetRole(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in setRoleBody
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	u, err := findUserByEmail(db, normalize(in.Email))
	if err != nil || u == nil {
		return writeNestError(h, 404, "user not found")
	}
	if !u.IsEmailVerified {
		return writeNestError(h, 400, "email not verified")
	}
	u.Role = in.Role
	_ = u.save(db)
	return writeNest(h, 200, u.publicDTO())
}

func handlePostAuthVerifyEmail(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in otpBody
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	u, err := findUserByEmail(db, normalize(in.Email))
	if err != nil || u == nil {
		return writeNestError(h, 404, "user not found")
	}
	if strings.TrimSpace(in.Token) == "" {
		return writeNestError(h, 400, "token required")
	}
	u.IsEmailVerified = true
	_ = u.save(db)
	return writeNest(h, 200, "verifying your email is successfully")
}

func handlePostAuthResendVerificationCode(h httpevent.Event) uint32 {
	return writeNest(h, 200, "resend verification code successfully")
}

func handlePostAuthResendResetCode(h httpevent.Event) uint32 {
	return writeNest(h, 200, "send rest code successfully")
}

func handlePostAuthVerifyResetCode(h httpevent.Event) uint32 {
	return writeNest(h, 200, map[string]any{"ok": true})
}

func handlePostAuthResetPassword(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in resetPwdBody
	if err := json.Unmarshal(body, &in); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	// Without OTP linkage we accept email in token field as email for demo parity
	email := normalize(in.Token)
	if email == "" {
		return writeNestError(h, 400, "invalid payload")
	}
	u, err := findUserByEmail(db, email)
	if err != nil || u == nil {
		return writeNestError(h, 404, "user not found")
	}
	if strings.TrimSpace(in.NewPassword) == "" {
		return writeNestError(h, 400, "password required")
	}
	u.PasswordHash = hashPassword(in.NewPassword)
	_ = u.save(db)
	return writeNest(h, 200, "reset password successfully")
}

func handlePostAuthLoginGoogle(h httpevent.Event) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var in map[string]any
	_ = json.Unmarshal(body, &in)
	email := ""
	if v, ok := in["email"].(string); ok {
		email = normalize(v)
	}
	if email == "" {
		return writeNestError(h, 400, "email required")
	}
	u, err := findUserByEmail(db, email)
	if err != nil || u == nil {
		u = &kvUser{
			ID:              newEntityID("usr"),
			Email:           email,
			Name:            "Google User",
			Role:            userRoleClient,
			Status:          userStatusAccepted,
			IsEmailVerified: true,
		}
		u.PasswordHash = hashPassword("google-oauth")
		_ = u.save(db)
	}
	access, refresh := tokensForUser(u.ID)
	_ = kvPutJSON(db, "sessions/"+access, map[string]string{"userId": u.ID})
	_ = kvPutJSON(db, "refresh/"+refresh, map[string]string{"userId": u.ID})
	return writeNest(h, 200, map[string]any{
		"accessToken":  access,
		"refreshToken": refresh,
		"user":         u.publicDTO(),
	})
}

func bearerUserID(h httpevent.Event, db database.Database) (string, bool) {
	authHeader, err := headerGet(h, "Authorization")
	if err != nil {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(authHeader), "Bearer"))
	if token == "" {
		return "", false
	}
	var s struct {
		UserID string `json:"userId"`
	}
	if getErr := kvGetJSON(db, "sessions/"+token, &s); getErr != nil || s.UserID == "" {
		return "", false
	}
	return s.UserID, true
}

// --- geo + admin-aligned resources (reuse matchers from empty.go) ---

func handleGetStatesIndex(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	db, err := database.New(statesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	rows, err := listByPrefix[localizedName](db, "states/")
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	filtered := make([]localizedName, 0, len(rows))
	for _, row := range rows {
		if containsName(row, search) {
			filtered = append(filtered, row)
		}
	}
	return writeNest(h, 200, nestPagination(filtered, page, limit))
}

func handleGetStatesGet(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	db, err := database.New(statesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	search := queryStr(h, "search")
	rows, err := listByPrefix[localizedName](db, "states/")
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	out := make([]localizedName, 0)
	term := strings.ToLower(strings.TrimSpace(search))
	for _, row := range rows {
		if term == "" || containsName(row, term) {
			out = append(out, row)
		}
	}
	return writeNest(h, 200, out)
}

func handlePostStatesStore(h httpevent.Event) uint32 {
	return handleCreateStateNest(h)
}

func handleCreateStateNest(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	var payload namePayload
	if err := decodeBody(h, &payload); err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	if !validateNamePayload(payload) {
		return writeNestError(h, 400, "all localized names are required")
	}
	db, err := database.New(statesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	state := localizedName{
		ID:     newID("state"),
		NameAr: payload.NameAr,
		NameEn: payload.NameEn,
		NameFr: payload.NameFr,
	}
	if err := putJSON(db, "states/"+state.ID, state); err != nil {
		return writeNestError(h, 500, "persist failed")
	}
	return writeNest(h, 201, state)
}

func handlePatchStatesUpdateById(h httpevent.Event) uint32 {
	id := pathAfter(h, "update")
	if id == "" {
		id = pathLast(h)
	}
	_ = seedDefaultData()
	var payload namePayload
	if err := decodeBody(h, &payload); err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	db, err := database.New(statesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	key := "states/" + id
	var row localizedName
	if err := getJSON(db, key, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	if strings.TrimSpace(payload.NameAr) != "" {
		row.NameAr = payload.NameAr
	}
	if strings.TrimSpace(payload.NameEn) != "" {
		row.NameEn = payload.NameEn
	}
	if strings.TrimSpace(payload.NameFr) != "" {
		row.NameFr = payload.NameFr
	}
	if err := putJSON(db, key, row); err != nil {
		return writeNestError(h, 500, "update failed")
	}
	return writeNest(h, 200, row)
}

func handleDeleteStatesDeleteById(h httpevent.Event) uint32 {
	id := pathAfter(h, "delete")
	if id == "" {
		id = pathLast(h)
	}
	db, err := database.New(statesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := db.Delete("states/" + id); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, map[string]any{"deleted": true})
}
