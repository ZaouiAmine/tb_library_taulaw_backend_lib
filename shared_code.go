// Taubyte library — single-file layout per .cursor/taubyte-folder-examples.md (Library).
package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/taubyte/go-sdk/database"
	"github.com/taubyte/go-sdk/event"
	httpevent "github.com/taubyte/go-sdk/http/event"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// kvUser mirrors essential fields from the Nest user entity for API responses.
type kvUser struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	Address            string `json:"address"`
	Role               int    `json:"role"`
	Status             int    `json:"status"`
	ImgURL             string `json:"imgUrl,omitempty"`
	ImageCover         string `json:"imageCover,omitempty"`
	Description        string `json:"description,omitempty"`
	IsEmailVerified    bool   `json:"isEmailVerified"`
	CreatedByLawyerID  string `json:"createdByLawyerId,omitempty"`
	VerificationStatus int    `json:"verificationStatus"`
	PasswordHash       string `json:"-"`
	RefreshToken       string `json:"-"`
	FcmToken           string `json:"-"`
}

func (u kvUser) publicDTO() map[string]any {
	return map[string]any{
		"id":                 u.ID,
		"name":               u.Name,
		"email":              u.Email,
		"phone":              u.Phone,
		"address":            u.Address,
		"role":               u.Role,
		"status":             u.Status,
		"imgUrl":             u.ImgURL,
		"imageCover":         u.ImageCover,
		"description":        u.Description,
		"isEmailVerified":    u.IsEmailVerified,
		"createdByLawyerId":  u.CreatedByLawyerID,
		"verificationStatus": u.VerificationStatus,
	}
}

type registerBody struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	Name             string `json:"name"`
	Phone            string `json:"phone"`
	StateID          string `json:"stateId"`
	SpecializationID string `json:"specializationId"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FcmToken string `json:"fcmToken"`
}

type refreshBody struct {
	RefreshToken string `json:"refreshToken"`
}

type otpBody struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type setRoleBody struct {
	Email string `json:"email"`
	Role  int    `json:"role"`
}

type resetPwdBody struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

const taulawKVMatcher = "taulaw_kv"

func writeNest(h httpevent.Event, httpStatus int, payload any) uint32 {
	type envelope struct {
		Code     int `json:"code"`
		Response any `json:"response"`
	}
	body, err := json.Marshal(envelope{Code: httpStatus, Response: payload})
	if err != nil {
		h.Write([]byte(`{"code":500,"response":"serialization failed"}`))
		h.Return(500)
		return 0
	}
	h.Headers().Set("Content-Type", "application/json")
	h.Write(body)
	h.Return(httpStatus)
	return 0
}

func writeNestError(h httpevent.Event, httpStatus int, message string) uint32 {
	return writeNest(h, httpStatus, map[string]any{"message": message})
}

// headerGet wraps Headers().Get so call sites satisfy Taubyte analysis (two-value return).
func headerGet(h httpevent.Event, key string) (string, error) {
	return h.Headers().Get(key)
}

// queryGet wraps Query().Get for the same reason as headerGet.
func queryGet(h httpevent.Event, key string) (string, error) {
	return h.Query().Get(key)
}

func queryStr(h httpevent.Event, key string) string {
	v, err := queryGet(h, key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func queryInt(h httpevent.Event, key string, def int) int {
	s := queryStr(h, key)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func readQueryPagination(h httpevent.Event) (page, limit int, search string) {
	page = queryInt(h, "page", 1)
	limit = queryInt(h, "limit", 10)
	search = strings.ToLower(strings.TrimSpace(queryStr(h, "search")))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit, search
}

func pathSegments(h httpevent.Event) []string {
	p, err := h.Path()
	if err != nil {
		return nil
	}
	p = strings.TrimSpace(p)
	if p == "" {
		return nil
	}
	p = path.Clean(p)
	return strings.Split(strings.Trim(p, "/"), "/")
}

// pathAfter returns the path segment immediately following marker (e.g. "update" -> id for .../update/:id).
func pathAfter(h httpevent.Event, marker string) string {
	segs := pathSegments(h)
	for i := 0; i < len(segs)-1; i++ {
		if segs[i] == marker {
			return segs[i+1]
		}
	}
	return ""
}

func pathLast(h httpevent.Event) string {
	segs := pathSegments(h)
	if len(segs) == 0 {
		return ""
	}
	return segs[len(segs)-1]
}

func nestPagination[T any](items []T, page, limit int) map[string]any {
	total := len(items)
	totalPages := 1
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
		if totalPages < 1 {
			totalPages = 1
		}
	}
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return map[string]any{
		"totalPages":  totalPages,
		"totalItems":  total,
		"data":        items[start:end],
		"currentPage": page,
	}
}

func kvDB() (database.Database, error) {
	return database.New(taulawKVMatcher)
}

func kvPutJSON(db database.Database, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return db.Put(key, b)
}

func kvGetJSON(db database.Database, key string, out any) error {
	b, err := db.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func kvDelete(db database.Database, key string) error {
	return db.Delete(key)
}

func kvListKeys(db database.Database, prefix string) ([]string, error) {
	return db.List(prefix)
}

func newEntityID(prefix string) string {
	return prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func localizedIndex(h httpevent.Event, matcher, keyPrefix, label string) uint32 {
	_ = seedDefaultData()
	db, err := database.New(matcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	rows, err := listByPrefix[localizedName](db, keyPrefix)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	filtered := make([]localizedName, 0, len(rows))
	for _, row := range rows {
		if containsName(row, search) {
			filtered = append(filtered, row)
		}
	}
	_ = label
	return writeNest(h, 200, nestPagination(filtered, page, limit))
}

func localizedGetPublic(h httpevent.Event, matcher, keyPrefix string) uint32 {
	_ = seedDefaultData()
	db, err := database.New(matcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	search := queryStr(h, "search")
	rows, err := listByPrefix[localizedName](db, keyPrefix)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	term := strings.ToLower(strings.TrimSpace(search))
	out := make([]localizedName, 0)
	for _, row := range rows {
		if term == "" || containsName(row, term) {
			out = append(out, row)
		}
	}
	return writeNest(h, 200, out)
}

func localizedStore(h httpevent.Event, matcher, keyPrefix, idPrefix string) uint32 {
	_ = seedDefaultData()
	var payload namePayload
	if err := decodeBody(h, &payload); err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	if !validateNamePayload(payload) {
		return writeNestError(h, 400, "all localized names are required")
	}
	db, err := database.New(matcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	row := localizedName{
		ID:     newID(idPrefix),
		NameAr: payload.NameAr,
		NameEn: payload.NameEn,
		NameFr: payload.NameFr,
	}
	if err := putJSON(db, keyPrefix+row.ID, row); err != nil {
		return writeNestError(h, 500, "persist failed")
	}
	return writeNest(h, 201, row)
}

func localizedPatch(h httpevent.Event, matcher, keyPrefix, afterMarker string) uint32 {
	id := pathAfter(h, afterMarker)
	if id == "" {
		id = pathLast(h)
	}
	_ = seedDefaultData()
	var payload namePayload
	if err := decodeBody(h, &payload); err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	db, err := database.New(matcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	key := keyPrefix + id
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

func localizedDelete(h httpevent.Event, matcher, keyPrefix, afterMarker string) uint32 {
	id := pathAfter(h, afterMarker)
	if id == "" {
		id = pathLast(h)
	}
	db, err := database.New(matcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := db.Delete(keyPrefix + id); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, map[string]any{"deleted": true})
}

func docKey(collection, id string) string {
	return collection + "/" + id
}

func docPut(db database.Database, collection, id string, doc map[string]any) error {
	if doc == nil {
		doc = map[string]any{}
	}
	return kvPutJSON(db, docKey(collection, id), doc)
}

func docGet(db database.Database, collection, id string) (map[string]any, error) {
	var out map[string]any
	err := kvGetJSON(db, docKey(collection, id), &out)
	return out, err
}

func docDelete(db database.Database, collection, id string) error {
	return kvDelete(db, docKey(collection, id))
}

func docList(db database.Database, collection string) ([]map[string]any, error) {
	keys, err := kvListKeys(db, collection+"/")
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		var row map[string]any
		if err := kvGetJSON(db, k, &row); err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func rawJSON(data []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func grList(h httpevent.Event, coll string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	docs, err := docList(db, coll)
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	if search != "" {
		filtered := make([]map[string]any, 0)
		for _, d := range docs {
			b, _ := json.Marshal(d)
			if strings.Contains(strings.ToLower(string(b)), search) {
				filtered = append(filtered, d)
			}
		}
		docs = filtered
	}
	return writeNest(h, 200, nestPagination(docs, page, limit))
}

func grGetByID(h httpevent.Event, coll string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	id := pathLast(h)
	doc, err := docGet(db, coll, id)
	if err != nil || doc == nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, doc)
}

func grPostJSON(h httpevent.Event, coll string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	doc := rawJSON(body)
	idStr := ""
	if v, ok := doc["id"].(string); ok {
		idStr = v
	}
	if idStr == "" {
		idStr = newEntityID(strings.ReplaceAll(coll, "/", "_"))
		doc["id"] = idStr
	}
	if err := docPut(db, coll, idStr, doc); err != nil {
		return writeNestError(h, 500, "persist failed")
	}
	return writeNest(h, 201, doc)
}

func grPatchJSON(h httpevent.Event, coll string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	id := pathLast(h)
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	patch := rawJSON(body)
	base, err := docGet(db, coll, id)
	if err != nil || base == nil {
		base = map[string]any{"id": id}
	}
	for k, v := range patch {
		base[k] = v
	}
	if err := docPut(db, coll, id, base); err != nil {
		return writeNestError(h, 500, "update failed")
	}
	return writeNest(h, 200, base)
}

func grDelete(h httpevent.Event, coll string) uint32 {
	db, err := kvDB()
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	id := pathLast(h)
	if err := docDelete(db, coll, id); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, map[string]any{"deleted": true})
}

func grOKMap(h httpevent.Event, m map[string]any) uint32 {
	return writeNest(h, 200, m)
}

const (
	userRoleAdmin      = 1
	userRoleLawyer     = 2
	userRoleOfficer    = 3
	userRoleClient     = 4
	userStatusPending  = 0
	userStatusAccepted = 1
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
	if err == nil && u != nil {
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
		// Admin SPA expects envelope code 201 (see AuthContext).
		return writeNest(h, 201, resp)
	}

	// Seeded admin lives in admin_users DB, not in KV — same path as /auth/login for the admin site.
	_ = seedDefaultData()
	adminDB, admErr := database.New(adminUsersMatcher)
	if admErr != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	adminUsers, listErr := listByPrefix[adminUser](adminDB, "admin_users/")
	if listErr != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	wantHash := hashPassword(in.Password)
	var matched *adminUser
	for i := range adminUsers {
		if normalize(adminUsers[i].Email) == email && adminUsers[i].PasswordHash == wantHash {
			matched = &adminUsers[i]
			break
		}
	}
	if matched == nil {
		return writeNestError(h, 404, "user not found")
	}
	access, refresh := tokensForUser(matched.ID)
	_ = kvPutJSON(db, "sessions/"+access, map[string]string{"userId": matched.ID})
	_ = kvPutJSON(db, "refresh/"+refresh, map[string]string{"userId": matched.ID})
	resp := map[string]any{
		"accessToken":  access,
		"refreshToken": refresh,
		"user": map[string]any{
			"id":    matched.ID,
			"email": matched.Email,
			"name":  "Admin",
			"role":  "admin",
		},
	}
	return writeNest(h, 201, resp)
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
		"cases":         0,
		"consultations": 0,
		"applications":  0,
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

func handleGetBannersPublic(h httpevent.Event, activeOnly bool) uint32 {
	_ = seedDefaultData()
	db, err := database.New(bannersMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	rows, err := listByPrefix[bannerItem](db, "banners/")
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	out := make([]bannerItem, 0, len(rows))
	typeFilter := queryStr(h, "type")
	for _, row := range rows {
		if activeOnly && row.Status != "active" {
			continue
		}
		if typeFilter != "" && !strings.EqualFold(row.Type, typeFilter) {
			continue
		}
		out = append(out, row)
	}
	return writeNest(h, 200, out)
}

func handleBannerByID(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	id := pathLast(h)
	db, err := database.New(bannersMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	var row bannerItem
	if err := getJSON(db, "banners/"+id, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, row)
}

func handlePostBanner(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var row bannerItem
	if err := decodeJSONBytes(body, &row); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = newID("banner")
	}
	db, err := database.New(bannersMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := putJSON(db, "banners/"+row.ID, row); err != nil {
		return writeNestError(h, 500, "persist failed")
	}
	return writeNest(h, 201, row)
}

func handlePatchBanner(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	id := pathLast(h)
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	db, err := database.New(bannersMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	var row bannerItem
	if err := getJSON(db, "banners/"+id, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	var patch bannerItem
	if err := decodeJSONBytes(body, &patch); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	if strings.TrimSpace(patch.Image) != "" {
		row.Image = patch.Image
	}
	if strings.TrimSpace(patch.Link) != "" {
		row.Link = patch.Link
	}
	if strings.TrimSpace(patch.Status) != "" {
		row.Status = patch.Status
	}
	if strings.TrimSpace(patch.Type) != "" {
		row.Type = patch.Type
	}
	if err := putJSON(db, "banners/"+id, row); err != nil {
		return writeNestError(h, 500, "update failed")
	}
	return writeNest(h, 200, row)
}

func handleDeleteBanner(h httpevent.Event) uint32 {
	id := pathLast(h)
	db, err := database.New(bannersMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := db.Delete("banners/" + id); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, map[string]any{"deleted": true})
}

func decodeJSONBytes(b []byte, v any) error {
	return json.Unmarshal(b, v)
}

func handleListConsultationPackagesNest(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	db, err := database.New(consultationsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	rows, err := listByPrefix[consultationPackage](db, "consultation_packages/")
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	filtered := make([]consultationPackage, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Name), search) {
			filtered = append(filtered, row)
		}
	}
	return writeNest(h, 200, nestPagination(filtered, page, limit))
}

func handleConsultationPackageByID(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	id := pathLast(h)
	db, err := database.New(consultationsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	var row consultationPackage
	if err := getJSON(db, "consultation_packages/"+id, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, row)
}

func handlePostConsultationPackage(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var row consultationPackage
	if err := json.Unmarshal(body, &row); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = newID("consult")
	}
	db, err := database.New(consultationsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := putJSON(db, "consultation_packages/"+row.ID, row); err != nil {
		return writeNestError(h, 500, "persist failed")
	}
	return writeNest(h, 201, row)
}

func handlePatchConsultationPackage(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	id := pathLast(h)
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	db, err := database.New(consultationsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	var row consultationPackage
	if err := getJSON(db, "consultation_packages/"+id, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	var patch consultationPackage
	if err := json.Unmarshal(body, &patch); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	if patch.Name != "" {
		row.Name = patch.Name
	}
	if patch.NumberOfConsultations != 0 {
		row.NumberOfConsultations = patch.NumberOfConsultations
	}
	if patch.Price != 0 {
		row.Price = patch.Price
	}
	row.IsActive = patch.IsActive
	if err := putJSON(db, "consultation_packages/"+id, row); err != nil {
		return writeNestError(h, 500, "update failed")
	}
	return writeNest(h, 200, row)
}

func handleDeleteConsultationPackage(h httpevent.Event) uint32 {
	id := pathLast(h)
	db, err := database.New(consultationsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := db.Delete("consultation_packages/" + id); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, map[string]any{"deleted": true})
}

func handleListLawyerPackagesNest(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	db, err := database.New(lawyerPackagesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	rows, err := listByPrefix[lawyerPackage](db, "lawyer_packages/")
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	filtered := make([]lawyerPackage, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Name), search) {
			filtered = append(filtered, row)
		}
	}
	return writeNest(h, 200, nestPagination(filtered, page, limit))
}

func handleLawyerPackageByID(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	id := pathLast(h)
	db, err := database.New(lawyerPackagesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	var row lawyerPackage
	if err := getJSON(db, "lawyer_packages/"+id, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, row)
}

func handlePostLawyerPackage(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	var row lawyerPackage
	if err := json.Unmarshal(body, &row); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	if strings.TrimSpace(row.ID) == "" {
		row.ID = newID("pkg")
	}
	if strings.TrimSpace(row.CreatedAt) == "" {
		row.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	db, err := database.New(lawyerPackagesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := putJSON(db, "lawyer_packages/"+row.ID, row); err != nil {
		return writeNestError(h, 500, "persist failed")
	}
	return writeNest(h, 201, row)
}

func handlePatchLawyerPackage(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	id := pathLast(h)
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	defer h.Body().Close()
	db, err := database.New(lawyerPackagesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	var row lawyerPackage
	if err := getJSON(db, "lawyer_packages/"+id, &row); err != nil {
		return writeNestError(h, 404, "not found")
	}
	var patch lawyerPackage
	if err := json.Unmarshal(body, &patch); err != nil {
		return writeNestError(h, 400, "invalid json")
	}
	if patch.Name != "" {
		row.Name = patch.Name
	}
	if patch.NumberOfCases != 0 {
		row.NumberOfCases = patch.NumberOfCases
	}
	if patch.NumberOfAssistants != 0 {
		row.NumberOfAssistants = patch.NumberOfAssistants
	}
	if patch.Price != 0 {
		row.Price = patch.Price
	}
	if patch.DurationInDays != 0 {
		row.DurationInDays = patch.DurationInDays
	}
	row.IsActive = patch.IsActive
	if err := putJSON(db, "lawyer_packages/"+id, row); err != nil {
		return writeNestError(h, 500, "update failed")
	}
	return writeNest(h, 200, row)
}

func handleDeleteLawyerPackage(h httpevent.Event) uint32 {
	id := pathLast(h)
	db, err := database.New(lawyerPackagesMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	if err := db.Delete("lawyer_packages/" + id); err != nil {
		return writeNestError(h, 404, "not found")
	}
	return writeNest(h, 200, map[string]any{"deleted": true})
}

func handleGetAdminReportsNest(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	db, err := database.New(reportsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	page, limit, search := readQueryPagination(h)
	rows, err := listByPrefix[reportItem](db, "reports/")
	if err != nil {
		return writeNestError(h, 500, "list failed")
	}
	filtered := make([]reportItem, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Reason), search) {
			filtered = append(filtered, row)
		}
	}
	return writeNest(h, 200, nestPagination(filtered, page, limit))
}

func handlePostAdminReportsActionNest(h httpevent.Event) uint32 {
	_ = seedDefaultData()
	var payload reportActionPayload
	if err := decodeBody(h, &payload); err != nil {
		return writeNestError(h, 400, "invalid body")
	}
	if strings.TrimSpace(payload.ReportID) == "" {
		return writeNestError(h, 400, "reportId is required")
	}
	db, err := database.New(reportsMatcher)
	if err != nil {
		return writeNestError(h, 500, "database unavailable")
	}
	key := "reports/" + payload.ReportID
	var row reportItem
	if err := getJSON(db, key, &row); err != nil {
		return writeNestError(h, 404, "report not found")
	}
	switch payload.Action {
	case 1:
		row.Status = "accepted"
	case 2:
		row.Status = "rejected"
	default:
		row.Status = "reviewed"
	}
	row.UntilDate = payload.UntilDate
	if err := putJSON(db, key, row); err != nil {
		return writeNestError(h, 500, "failed to update report")
	}
	return writeNest(h, 200, row)
}

const (
	adminUsersMatcher      = "admin_users"
	statesMatcher          = "states"
	specializationsMatcher = "specializations"
	pendingLawyersMatcher  = "pending_lawyers"
	reportsMatcher         = "reports"
	caseCategoriesMatcher  = "case_categories"
	caseChambersMatcher    = "case_chambers"
	casePhasesMatcher      = "case_phases"
	bannersMatcher         = "banners"
	consultationsMatcher   = "consultation_packages"
	lawyerPackagesMatcher  = "lawyer_packages"
	subscriptionsMatcher   = "subscriptions"
	lawyerRequestsMatcher  = "lawyer_requests"
)

const (
	seedAdminID       = "admin_1"
	seedAdminEmail    = "admin@lawgen.com"
	seedAdminPassword = "admin123456"
	seedAdminRole     = "admin"
)

type localizedName struct {
	ID     string `json:"id"`
	NameAr string `json:"nameAr"`
	NameEn string `json:"nameEn"`
	NameFr string `json:"nameFr"`
}

type dashboardStats struct {
	AIPetitions                 int `json:"aiPetitions"`
	FormRequestsCountForLawyer  int `json:"formRequestsCountForLawyer"`
	PublishedCasesCount         int `json:"publishedCasesCount"`
	PublishedConsultationsCount int `json:"publishedConsultationsCount"`
	ConsultingTransactionsCount int `json:"consultingTransactionsCount"`
	CasesCount                  int `json:"casesCount"`
	FormRequestsCountForOfficer int `json:"formRequestsCountForOfficer"`
	PublicationsCount           int `json:"publicationsCount"`
	TotalLawyers                int `json:"totalLawyers"`
	PendingVerification         int `json:"pendingVerification"`
	OpenReports                 int `json:"openReports"`
	TotalStates                 int `json:"totalStates"`
}

type pendingLawyer struct {
	ID                 string `json:"id"`
	FullName           string `json:"fullName"`
	Email              string `json:"email"`
	VerificationStatus string `json:"verificationStatus"`
}

type reportItem struct {
	ID        string `json:"id"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	UntilDate string `json:"untilDate,omitempty"`
}

type bannerItem struct {
	ID     string `json:"id"`
	Image  string `json:"image"`
	Link   string `json:"link"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type consultationPackage struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	NumberOfConsultations int     `json:"numberOfConsultations"`
	Price                 float64 `json:"price"`
	IsActive              bool    `json:"isActive"`
}

type lawyerPackage struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	NumberOfCases      int     `json:"numberOfCases"`
	NumberOfAssistants int     `json:"numberOfAssistants"`
	Price              float64 `json:"price"`
	DurationInDays     int     `json:"durationInDays"`
	IsActive           bool    `json:"isActive"`
	CreatedAt          string  `json:"createdAt"`
}

type subscriptionItem struct {
	ID          string  `json:"id"`
	LawyerID    string  `json:"lawyerId"`
	LawyerName  string  `json:"lawyerName"`
	PackageID   string  `json:"packageId"`
	PackageName string  `json:"packageName"`
	StartDate   string  `json:"startDate"`
	EndDate     string  `json:"endDate"`
	Status      string  `json:"status"`
	Price       float64 `json:"price"`
	CreatedAt   string  `json:"createdAt"`
	Lawyer      struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"lawyer"`
	Package struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	} `json:"package"`
}

type lawyerRequestItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Address         string `json:"address"`
	Status          int    `json:"status"`
	Role            int    `json:"role"`
	IsEmailVerified bool   `json:"isEmailVerified"`
}

type pagedAnyResponse[T any] struct {
	Data        []T     `json:"data"`
	CurrentPage int     `json:"currentPage"`
	TotalPages  int     `json:"totalPages"`
	TotalItems  int     `json:"totalItems"`
	NextPageURL *string `json:"nextPageUrl"`
	PrevPageURL *string `json:"prevPageUrl"`
}

type adminUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	Role         string `json:"role"`
	CreatedAt    string `json:"createdAt"`
}

type namePayload struct {
	NameAr string `json:"nameAr"`
	NameEn string `json:"nameEn"`
	NameFr string `json:"nameFr"`
}

type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshPayload struct {
	RefreshToken string `json:"refreshToken"`
}

type reportActionPayload struct {
	ReportID  string `json:"reportId"`
	Action    int    `json:"action"`
	UntilDate string `json:"untilDate"`
}

type verifyLawyerPayload struct {
	LawyerID string `json:"lawyerId"`
}

type pagedLocalizedResponse struct {
	Data  []localizedName `json:"data"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Total int             `json:"total"`
}

type pagedPendingLawyersResponse struct {
	Data  []pendingLawyer `json:"data"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Total int             `json:"total"`
}

type pagedReportsResponse struct {
	Data  []reportItem `json:"data"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Total int          `json:"total"`
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func containsName(item localizedName, term string) bool {
	if term == "" {
		return true
	}

	return strings.Contains(normalize(item.NameAr), term) ||
		strings.Contains(normalize(item.NameEn), term) ||
		strings.Contains(normalize(item.NameFr), term)
}

func parsePagination(h httpevent.Event) (page int, limit int, search string) {
	_ = h
	page = 1
	limit = 10
	return page, limit, search
}

func writeJSON(h httpevent.Event, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		h.Write([]byte(`{"success":false,"message":"serialization failed"}`))
		h.Return(500)
		return
	}

	h.Headers().Set("Content-Type", "application/json")
	h.Write(body)
	h.Return(status)
}

func writeError(h httpevent.Event, status int, message string) {
	type errorResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	writeJSON(h, status, errorResponse{
		Success: false,
		Message: message,
	})
}

func decodeBody[T any](h httpevent.Event, payload *T) error {
	body, err := io.ReadAll(h.Body())
	if err != nil {
		return err
	}
	defer h.Body().Close()

	if len(body) == 0 {
		return io.EOF
	}

	return json.Unmarshal(body, payload)
}

func requireAdminAuth(h httpevent.Event) bool {
	_ = h
	return true
}

func validateNamePayload(input namePayload) bool {
	return strings.TrimSpace(input.NameAr) != "" &&
		strings.TrimSpace(input.NameEn) != "" &&
		strings.TrimSpace(input.NameFr) != ""
}

func paginateBounds(total, page, limit int) (start int, end int) {
	start = (page - 1) * limit
	if start > total {
		start = total
	}
	end = start + limit
	if end > total {
		end = total
	}
	return start, end
}

func buildPagedResponse[T any](items []T, page, limit int) pagedAnyResponse[T] {
	total := len(items)
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	if totalPages == 0 {
		totalPages = 1
	}
	start, end := paginateBounds(total, page, limit)
	resp := pagedAnyResponse[T]{
		Data:        items[start:end],
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalItems:  total,
	}
	if page < totalPages {
		next := "?page=" + strconv.Itoa(page+1)
		resp.NextPageURL = &next
	}
	if page > 1 && total > 0 {
		prev := "?page=" + strconv.Itoa(page-1)
		resp.PrevPageURL = &prev
	}
	return resp
}

func newID(prefix string) string {
	return prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func putJSON(db database.Database, key string, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Put(key, body)
}

func getJSON[T any](db database.Database, key string, out *T) error {
	body, err := db.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func listByPrefix[T any](db database.Database, prefix string) ([]T, error) {
	keys, err := db.List(prefix)
	if err != nil {
		return nil, err
	}

	sort.Strings(keys)
	out := make([]T, 0, len(keys))

	for _, key := range keys {
		body, getErr := db.Get(key)
		if getErr != nil {
			continue
		}

		var row T
		if unmarshalErr := json.Unmarshal(body, &row); unmarshalErr != nil {
			continue
		}
		out = append(out, row)
	}

	return out, nil
}

func seedDefaultData() error {
	adminDB, err := database.New(adminUsersMatcher)
	if err != nil {
		return err
	}
	statesDB, err := database.New(statesMatcher)
	if err != nil {
		return err
	}
	specializationsDB, err := database.New(specializationsMatcher)
	if err != nil {
		return err
	}
	pendingLawyersDB, err := database.New(pendingLawyersMatcher)
	if err != nil {
		return err
	}
	reportsDB, err := database.New(reportsMatcher)
	if err != nil {
		return err
	}
	caseCategoriesDB, err := database.New(caseCategoriesMatcher)
	if err != nil {
		return err
	}
	caseChambersDB, err := database.New(caseChambersMatcher)
	if err != nil {
		return err
	}
	casePhasesDB, err := database.New(casePhasesMatcher)
	if err != nil {
		return err
	}
	bannersDB, err := database.New(bannersMatcher)
	if err != nil {
		return err
	}
	consultationsDB, err := database.New(consultationsMatcher)
	if err != nil {
		return err
	}
	lawyerPackagesDB, err := database.New(lawyerPackagesMatcher)
	if err != nil {
		return err
	}
	subscriptionsDB, err := database.New(subscriptionsMatcher)
	if err != nil {
		return err
	}
	lawyerRequestsDB, err := database.New(lawyerRequestsMatcher)
	if err != nil {
		return err
	}

	seedAdmin := adminUser{
		ID:           seedAdminID,
		Email:        seedAdminEmail,
		PasswordHash: hashPassword(seedAdminPassword),
		Role:         seedAdminRole,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	adminKey := "admin_users/" + seedAdmin.ID
	if _, getErr := adminDB.Get(adminKey); getErr != nil {
		if putErr := putJSON(adminDB, adminKey, seedAdmin); putErr != nil {
			return putErr
		}
	}

	defaultStates := []localizedName{
		{ID: "state_1", NameAr: "الجزائر", NameEn: "Algiers", NameFr: "Alger"},
		{ID: "state_2", NameAr: "وهران", NameEn: "Oran", NameFr: "Oran"},
	}
	for _, row := range defaultStates {
		key := "states/" + row.ID
		if _, getErr := statesDB.Get(key); getErr != nil {
			if putErr := putJSON(statesDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultSpecializations := []localizedName{
		{ID: "spec_1", NameAr: "قانون مدني", NameEn: "Civil Law", NameFr: "Droit Civil"},
		{ID: "spec_2", NameAr: "قانون تجاري", NameEn: "Commercial Law", NameFr: "Droit Commercial"},
	}
	for _, row := range defaultSpecializations {
		key := "specializations/" + row.ID
		if _, getErr := specializationsDB.Get(key); getErr != nil {
			if putErr := putJSON(specializationsDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultPendingLawyers := []pendingLawyer{
		{ID: "lawyer_1", FullName: "Nadia Benali", Email: "nadia.benali@example.com", VerificationStatus: "pending"},
		{ID: "lawyer_2", FullName: "Yacine Bouchareb", Email: "yacine.bouchareb@example.com", VerificationStatus: "pending"},
	}
	for _, row := range defaultPendingLawyers {
		key := "pending_lawyers/" + row.ID
		if _, getErr := pendingLawyersDB.Get(key); getErr != nil {
			if putErr := putJSON(pendingLawyersDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultReports := []reportItem{
		{ID: "report_1", Reason: "Abusive comment", Status: "pending"},
		{ID: "report_2", Reason: "Fraud suspicion", Status: "pending"},
	}
	for _, row := range defaultReports {
		key := "reports/" + row.ID
		if _, getErr := reportsDB.Get(key); getErr != nil {
			if putErr := putJSON(reportsDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultCaseCategories := []localizedName{
		{ID: "category_1", NameAr: "قضايا الأسرة", NameEn: "Family Cases", NameFr: "Affaires Familiales"},
		{ID: "category_2", NameAr: "القضايا التجارية", NameEn: "Commercial Cases", NameFr: "Affaires Commerciales"},
		{ID: "category_3", NameAr: "القضايا الجنائية", NameEn: "Criminal Cases", NameFr: "Affaires Pénales"},
	}
	for _, row := range defaultCaseCategories {
		key := "case_categories/" + row.ID
		if _, getErr := caseCategoriesDB.Get(key); getErr != nil {
			if putErr := putJSON(caseCategoriesDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultCaseChambers := []localizedName{
		{ID: "chamber_1", NameAr: "الغرفة المدنية", NameEn: "Civil Chamber", NameFr: "Chambre Civile"},
		{ID: "chamber_2", NameAr: "الغرفة التجارية", NameEn: "Commercial Chamber", NameFr: "Chambre Commerciale"},
		{ID: "chamber_3", NameAr: "الغرفة الجزائية", NameEn: "Criminal Chamber", NameFr: "Chambre Pénale"},
	}
	for _, row := range defaultCaseChambers {
		key := "case_chambers/" + row.ID
		if _, getErr := caseChambersDB.Get(key); getErr != nil {
			if putErr := putJSON(caseChambersDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultCasePhases := []localizedName{
		{ID: "phase_1", NameAr: "إيداع الملف", NameEn: "Case Filing", NameFr: "Dépôt du dossier"},
		{ID: "phase_2", NameAr: "جلسة الاستماع", NameEn: "Hearing", NameFr: "Audience"},
		{ID: "phase_3", NameAr: "الحكم", NameEn: "Judgment", NameFr: "Jugement"},
	}
	for _, row := range defaultCasePhases {
		key := "case_phases/" + row.ID
		if _, getErr := casePhasesDB.Get(key); getErr != nil {
			if putErr := putJSON(casePhasesDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultBanners := []bannerItem{
		{ID: "banner_1", Image: "seed-banner-1.jpg", Link: "https://lawgen.app/promo/1", Status: "active", Type: "lawyer"},
		{ID: "banner_2", Image: "seed-banner-2.jpg", Link: "https://lawgen.app/promo/2", Status: "active", Type: "client"},
	}
	for _, row := range defaultBanners {
		key := "banners/" + row.ID
		if _, getErr := bannersDB.Get(key); getErr != nil {
			if putErr := putJSON(bannersDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultConsultations := []consultationPackage{
		{ID: "consult_1", Name: "Basic Consultation", NumberOfConsultations: 3, Price: 49.99, IsActive: true},
		{ID: "consult_2", Name: "Business Consultation", NumberOfConsultations: 8, Price: 149.00, IsActive: true},
		{ID: "consult_3", Name: "Premium Consultation", NumberOfConsultations: 15, Price: 249.00, IsActive: true},
	}
	for _, row := range defaultConsultations {
		key := "consultation_packages/" + row.ID
		if _, getErr := consultationsDB.Get(key); getErr != nil {
			if putErr := putJSON(consultationsDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	now := time.Now().UTC()
	defaultLawyerPackages := []lawyerPackage{
		{ID: "pkg_1", Name: "Starter", NumberOfCases: 20, NumberOfAssistants: 1, Price: 39.00, DurationInDays: 30, IsActive: true, CreatedAt: now.AddDate(0, -2, 0).Format(time.RFC3339)},
		{ID: "pkg_2", Name: "Growth", NumberOfCases: 60, NumberOfAssistants: 3, Price: 99.00, DurationInDays: 90, IsActive: true, CreatedAt: now.AddDate(0, -1, 0).Format(time.RFC3339)},
		{ID: "pkg_3", Name: "Enterprise", NumberOfCases: 200, NumberOfAssistants: 10, Price: 249.00, DurationInDays: 365, IsActive: true, CreatedAt: now.AddDate(0, 0, -15).Format(time.RFC3339)},
	}
	for _, row := range defaultLawyerPackages {
		key := "lawyer_packages/" + row.ID
		if _, getErr := lawyerPackagesDB.Get(key); getErr != nil {
			if putErr := putJSON(lawyerPackagesDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultLawyerRequests := []lawyerRequestItem{
		{ID: "request_1", Name: "Karim Saadi", Email: "karim.saadi@example.com", Phone: "+213550001111", Address: "Algiers", Status: 0, Role: 2, IsEmailVerified: true},
		{ID: "request_2", Name: "Amina Hadj", Email: "amina.hadj@example.com", Phone: "+213550002222", Address: "Oran", Status: 0, Role: 2, IsEmailVerified: true},
		{ID: "request_3", Name: "Sofiane Rahal", Email: "sofiane.rahal@example.com", Phone: "+213550003333", Address: "Constantine", Status: 1, Role: 2, IsEmailVerified: true},
	}
	for _, row := range defaultLawyerRequests {
		key := "lawyer_requests/" + row.ID
		if _, getErr := lawyerRequestsDB.Get(key); getErr != nil {
			if putErr := putJSON(lawyerRequestsDB, key, row); putErr != nil {
				return putErr
			}
		}
	}

	defaultSubscriptions := []subscriptionItem{
		{
			ID: "sub_1", LawyerID: "request_1", LawyerName: "Karim Saadi", PackageID: "pkg_2", PackageName: "Growth",
			StartDate: now.AddDate(0, -1, 0).Format(time.RFC3339), EndDate: now.AddDate(0, 2, 0).Format(time.RFC3339),
			Status: "active", Price: 99.00, CreatedAt: now.AddDate(0, -1, 0).Format(time.RFC3339),
		},
		{
			ID: "sub_2", LawyerID: "request_2", LawyerName: "Amina Hadj", PackageID: "pkg_1", PackageName: "Starter",
			StartDate: now.AddDate(0, -3, 0).Format(time.RFC3339), EndDate: now.AddDate(0, -1, -5).Format(time.RFC3339),
			Status: "expired", Price: 39.00, CreatedAt: now.AddDate(0, -3, 0).Format(time.RFC3339),
		},
	}
	for i := range defaultSubscriptions {
		defaultSubscriptions[i].Lawyer = struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		}{
			ID:    defaultSubscriptions[i].LawyerID,
			Name:  defaultSubscriptions[i].LawyerName,
			Email: defaultSubscriptions[i].LawyerID + "@example.com",
			Phone: "+213000000000",
		}
		defaultSubscriptions[i].Package = struct {
			ID    string  `json:"id"`
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}{
			ID:    defaultSubscriptions[i].PackageID,
			Name:  defaultSubscriptions[i].PackageName,
			Price: defaultSubscriptions[i].Price,
		}
		key := "subscriptions/" + defaultSubscriptions[i].ID
		if _, getErr := subscriptionsDB.Get(key); getErr != nil {
			if putErr := putJSON(subscriptionsDB, key, defaultSubscriptions[i]); putErr != nil {
				return putErr
			}
		}
	}

	return nil
}

//export LoginAdmin
func LoginAdmin(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}

	if seedErr := seedDefaultData(); seedErr != nil {
		writeError(h, 500, "failed to initialize admin data")
		return 0
	}

	var input loginPayload
	if decodeErr := decodeBody(h, &input); decodeErr != nil {
		writeError(h, 400, "invalid login payload")
		return 0
	}

	email := normalize(input.Email)
	password := strings.TrimSpace(input.Password)
	if email == "" || password == "" {
		writeError(h, 400, "email and password are required")
		return 0
	}

	adminDB, dbErr := database.New(adminUsersMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access admin users database")
		return 0
	}

	adminUsers, listErr := listByPrefix[adminUser](adminDB, "admin_users/")
	if listErr != nil {
		writeError(h, 500, "failed to load admin users")
		return 0
	}

	passwordHash := hashPassword(password)
	var matched *adminUser
	for i := range adminUsers {
		if normalize(adminUsers[i].Email) == email && adminUsers[i].PasswordHash == passwordHash {
			matched = &adminUsers[i]
			break
		}
	}

	if matched == nil {
		writeError(h, 401, "invalid email or password")
		return 0
	}

	seed := strconv.FormatInt(time.Now().UnixNano(), 10)
	response := map[string]any{
		"code":    201,
		"success": true,
		"response": map[string]any{
			"accessToken":  "access_" + seed,
			"refreshToken": "refresh_" + seed,
			"user": map[string]any{
				"id":    matched.ID,
				"email": matched.Email,
				"name":  "Admin",
				"role":  matched.Role,
			},
		},
		"data": map[string]any{
			"accessToken":  "access_" + seed,
			"refreshToken": "refresh_" + seed,
			"user": map[string]any{
				"id":    matched.ID,
				"email": matched.Email,
				"name":  "Admin",
				"role":  matched.Role,
			},
		},
	}

	writeJSON(h, 201, response)
	return 0
}

//export RefreshAdminToken
func RefreshAdminToken(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}

	var input refreshPayload
	if decodeErr := decodeBody(h, &input); decodeErr != nil {
		writeError(h, 400, "invalid refresh payload")
		return 0
	}

	if strings.TrimSpace(input.RefreshToken) == "" {
		writeError(h, 400, "refresh token is required")
		return 0
	}

	access := "access_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	writeJSON(h, 201, map[string]any{
		"code":    201,
		"success": true,
		"response": map[string]any{
			"accessToken": access,
		},
		"data": map[string]any{
			"accessToken": access,
		},
	})

	return 0
}

//export ListStates
func ListStates(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	db, dbErr := database.New(statesMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access states database")
		return 0
	}

	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[localizedName](db, "states/")
	if listErr != nil {
		writeError(h, 500, "failed to list states")
		return 0
	}

	filtered := make([]localizedName, 0, len(rows))
	for _, row := range rows {
		if containsName(row, search) {
			filtered = append(filtered, row)
		}
	}

	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export CreateState
func CreateState(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	var payload namePayload
	if decodeErr := decodeBody(h, &payload); decodeErr != nil {
		writeError(h, 400, "invalid state payload")
		return 0
	}
	if !validateNamePayload(payload) {
		writeError(h, 400, "all localized names are required")
		return 0
	}

	db, dbErr := database.New(statesMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access states database")
		return 0
	}

	state := localizedName{
		ID:     newID("state"),
		NameAr: payload.NameAr,
		NameEn: payload.NameEn,
		NameFr: payload.NameFr,
	}
	if putErr := putJSON(db, "states/"+state.ID, state); putErr != nil {
		writeError(h, 500, "failed to persist state")
		return 0
	}

	writeJSON(h, 201, state)
	return 0
}

//export ListSpecializations
func ListSpecializations(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	db, dbErr := database.New(specializationsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access specializations database")
		return 0
	}

	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[localizedName](db, "specializations/")
	if listErr != nil {
		writeError(h, 500, "failed to list specializations")
		return 0
	}

	filtered := make([]localizedName, 0, len(rows))
	for _, row := range rows {
		if containsName(row, search) {
			filtered = append(filtered, row)
		}
	}

	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export CreateSpecialization
func CreateSpecialization(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	var payload namePayload
	if decodeErr := decodeBody(h, &payload); decodeErr != nil {
		writeError(h, 400, "invalid specialization payload")
		return 0
	}
	if !validateNamePayload(payload) {
		writeError(h, 400, "all localized names are required")
		return 0
	}

	db, dbErr := database.New(specializationsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access specializations database")
		return 0
	}

	row := localizedName{
		ID:     newID("spec"),
		NameAr: payload.NameAr,
		NameEn: payload.NameEn,
		NameFr: payload.NameFr,
	}
	if putErr := putJSON(db, "specializations/"+row.ID, row); putErr != nil {
		writeError(h, 500, "failed to persist specialization")
		return 0
	}

	writeJSON(h, 201, row)
	return 0
}

//export ListPendingLawyers
func ListPendingLawyers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	db, dbErr := database.New(pendingLawyersMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access pending lawyers database")
		return 0
	}

	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[pendingLawyer](db, "pending_lawyers/")
	if listErr != nil {
		writeError(h, 500, "failed to list pending lawyers")
		return 0
	}

	filtered := make([]pendingLawyer, 0, len(rows))
	for _, row := range rows {
		if row.VerificationStatus != "pending" {
			continue
		}
		if search == "" || strings.Contains(normalize(row.FullName), search) || strings.Contains(normalize(row.Email), search) {
			filtered = append(filtered, row)
		}
	}

	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export VerifyPendingLawyer
func VerifyPendingLawyer(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	var payload verifyLawyerPayload
	decodeErr := decodeBody(h, &payload)
	if decodeErr != nil && decodeErr != io.EOF {
		writeError(h, 400, "invalid lawyer verification payload")
		return 0
	}

	if strings.TrimSpace(payload.LawyerID) == "" {
		writeError(h, 400, "lawyerId is required")
		return 0
	}

	db, dbErr := database.New(pendingLawyersMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access pending lawyers database")
		return 0
	}

	key := "pending_lawyers/" + payload.LawyerID
	var row pendingLawyer
	if getErr := getJSON(db, key, &row); getErr != nil {
		writeError(h, 404, "lawyer not found")
		return 0
	}

	row.VerificationStatus = "accepted"
	if putErr := putJSON(db, key, row); putErr != nil {
		writeError(h, 500, "failed to update lawyer verification status")
		return 0
	}

	writeJSON(h, 200, row)
	return 0
}

//export GetDashboardStatus
func GetDashboardStatus(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	statesDB, stateErr := database.New(statesMatcher)
	if stateErr != nil {
		writeError(h, 500, "failed to access states database")
		return 0
	}
	pendingDB, pendingErr := database.New(pendingLawyersMatcher)
	if pendingErr != nil {
		writeError(h, 500, "failed to access pending lawyers database")
		return 0
	}
	reportsDB, reportsErr := database.New(reportsMatcher)
	if reportsErr != nil {
		writeError(h, 500, "failed to access reports database")
		return 0
	}

	statesRows, _ := listByPrefix[localizedName](statesDB, "states/")
	pendingRows, _ := listByPrefix[pendingLawyer](pendingDB, "pending_lawyers/")
	reportRows, _ := listByPrefix[reportItem](reportsDB, "reports/")
	caseRows := make([]localizedName, 0)
	consultRows := make([]consultationPackage, 0)
	subscriptionRows := make([]subscriptionItem, 0)

	if caseDB, dbErr := database.New(caseCategoriesMatcher); dbErr == nil {
		caseRows, _ = listByPrefix[localizedName](caseDB, "case_categories/")
	}
	if consultDB, dbErr := database.New(consultationsMatcher); dbErr == nil {
		consultRows, _ = listByPrefix[consultationPackage](consultDB, "consultation_packages/")
	}
	if subDB, dbErr := database.New(subscriptionsMatcher); dbErr == nil {
		subscriptionRows, _ = listByPrefix[subscriptionItem](subDB, "subscriptions/")
	}

	pendingCount := 0
	for _, row := range pendingRows {
		if row.VerificationStatus == "pending" {
			pendingCount++
		}
	}

	openReports := 0
	for _, row := range reportRows {
		if row.Status == "pending" {
			openReports++
		}
	}

	stats := dashboardStats{
		AIPetitions:                 len(reportRows) + len(caseRows),
		FormRequestsCountForLawyer:  len(pendingRows),
		PublishedCasesCount:         len(caseRows),
		PublishedConsultationsCount: len(consultRows),
		ConsultingTransactionsCount: len(subscriptionRows),
		CasesCount:                  len(caseRows),
		FormRequestsCountForOfficer: len(statesRows),
		PublicationsCount:           len(caseRows) + len(consultRows),
		TotalLawyers:                len(pendingRows),
		PendingVerification:         pendingCount,
		OpenReports:                 openReports,
		TotalStates:                 len(statesRows),
	}
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": stats,
		"data":     stats,
	})

	return 0
}

//export ListReports
func ListReports(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	db, dbErr := database.New(reportsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access reports database")
		return 0
	}

	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[reportItem](db, "reports/")
	if listErr != nil {
		writeError(h, 500, "failed to list reports")
		return 0
	}

	filtered := make([]reportItem, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Reason), search) {
			filtered = append(filtered, row)
		}
	}

	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ReportAction
func ReportAction(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}

	var payload reportActionPayload
	if decodeErr := decodeBody(h, &payload); decodeErr != nil {
		writeError(h, 400, "invalid report action payload")
		return 0
	}
	if strings.TrimSpace(payload.ReportID) == "" {
		writeError(h, 400, "reportId is required")
		return 0
	}

	db, dbErr := database.New(reportsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access reports database")
		return 0
	}

	key := "reports/" + payload.ReportID
	var row reportItem
	if getErr := getJSON(db, key, &row); getErr != nil {
		writeError(h, 404, "report not found")
		return 0
	}

	switch payload.Action {
	case 1:
		row.Status = "accepted"
	case 2:
		row.Status = "rejected"
	default:
		row.Status = "reviewed"
	}
	row.UntilDate = payload.UntilDate

	if putErr := putJSON(db, key, row); putErr != nil {
		writeError(h, 500, "failed to update report")
		return 0
	}

	writeJSON(h, 200, row)
	return 0
}

func listLocalizedModule(h httpevent.Event, matcher, prefix, errLabel string) uint32 {
	db, dbErr := database.New(matcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access "+errLabel+" database")
		return 0
	}
	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[localizedName](db, prefix)
	if listErr != nil {
		writeError(h, 500, "failed to list "+errLabel)
		return 0
	}
	filtered := make([]localizedName, 0, len(rows))
	for _, row := range rows {
		if containsName(row, search) {
			filtered = append(filtered, row)
		}
	}
	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ListCaseCategories
func ListCaseCategories(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	return listLocalizedModule(h, caseCategoriesMatcher, "case_categories/", "case categories")
}

//export ListCaseChambers
func ListCaseChambers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	return listLocalizedModule(h, caseChambersMatcher, "case_chambers/", "case chambers")
}

//export ListCasePhases
func ListCasePhases(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	return listLocalizedModule(h, casePhasesMatcher, "case_phases/", "case phases")
}

//export ListBanners
func ListBanners(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(bannersMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access banners database")
		return 0
	}
	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[bannerItem](db, "banners/")
	if listErr != nil {
		writeError(h, 500, "failed to list banners")
		return 0
	}
	filtered := make([]bannerItem, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Link), search) || strings.Contains(normalize(row.Type), search) {
			filtered = append(filtered, row)
		}
	}
	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ListConsultationPackages
func ListConsultationPackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(consultationsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access consultation packages database")
		return 0
	}
	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[consultationPackage](db, "consultation_packages/")
	if listErr != nil {
		writeError(h, 500, "failed to list consultation packages")
		return 0
	}
	filtered := make([]consultationPackage, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Name), search) {
			filtered = append(filtered, row)
		}
	}
	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ListLawyerPackages
func ListLawyerPackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(lawyerPackagesMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access lawyer packages database")
		return 0
	}
	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[lawyerPackage](db, "lawyer_packages/")
	if listErr != nil {
		writeError(h, 500, "failed to list lawyer packages")
		return 0
	}
	filtered := make([]lawyerPackage, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Name), search) {
			filtered = append(filtered, row)
		}
	}
	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ListSubscriptionHistory
func ListSubscriptionHistory(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(subscriptionsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access subscriptions database")
		return 0
	}
	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[subscriptionItem](db, "subscriptions/")
	if listErr != nil {
		writeError(h, 500, "failed to list subscriptions")
		return 0
	}
	filtered := make([]subscriptionItem, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.LawyerName), search) || strings.Contains(normalize(row.PackageName), search) {
			filtered = append(filtered, row)
		}
	}
	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ListLawyerRequests
func ListLawyerRequests(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(lawyerRequestsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access lawyer requests database")
		return 0
	}
	page, limit, search := parsePagination(h)
	rows, listErr := listByPrefix[lawyerRequestItem](db, "lawyer_requests/")
	if listErr != nil {
		writeError(h, 500, "failed to list lawyer requests")
		return 0
	}
	filtered := make([]lawyerRequestItem, 0, len(rows))
	for _, row := range rows {
		if search == "" || strings.Contains(normalize(row.Name), search) || strings.Contains(normalize(row.Email), search) {
			filtered = append(filtered, row)
		}
	}
	resp := buildPagedResponse(filtered, page, limit)
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": resp,
		"data":     resp,
	})
	return 0
}

//export ListAllLawyers
func ListAllLawyers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(lawyerRequestsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access lawyer requests database")
		return 0
	}
	rows, listErr := listByPrefix[lawyerRequestItem](db, "lawyer_requests/")
	if listErr != nil {
		writeError(h, 500, "failed to list lawyers")
		return 0
	}
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": rows,
		"data":     rows,
	})
	return 0
}

//export AcceptLawyerRequest
func AcceptLawyerRequest(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	if !requireAdminAuth(h) {
		writeError(h, 401, "unauthorized")
		return 0
	}
	db, dbErr := database.New(lawyerRequestsMatcher)
	if dbErr != nil {
		writeError(h, 500, "failed to access lawyer requests database")
		return 0
	}

	var payload verifyLawyerPayload
	decodeErr := decodeBody(h, &payload)
	if decodeErr != nil && decodeErr != io.EOF {
		writeError(h, 400, "invalid payload")
		return 0
	}
	lawyerID := strings.TrimSpace(payload.LawyerID)
	if lawyerID == "" {
		writeError(h, 400, "lawyer id is required")
		return 0
	}

	key := "lawyer_requests/" + lawyerID
	var row lawyerRequestItem
	if getErr := getJSON(db, key, &row); getErr != nil {
		writeError(h, 404, "lawyer request not found")
		return 0
	}
	row.Status = 1
	if putErr := putJSON(db, key, row); putErr != nil {
		writeError(h, 500, "failed to update lawyer request")
		return 0
	}
	writeJSON(h, 200, map[string]any{
		"success":  true,
		"response": row,
		"data":     row,
	})
	return 0
}

func routeDeleteAssistants(h httpevent.Event) uint32 {
	return grDelete(h, "assistants")
}

func routeDeleteBannersById(h httpevent.Event) uint32 {
	return handleDeleteBanner(h)
}

func routeDeleteCaseCategoriesDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, caseCategoriesMatcher, "case_categories/", "delete")
}

func routeDeleteCaseChambersDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, caseChambersMatcher, "case_chambers/", "delete")
}

func routeDeleteCasePhasesDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, casePhasesMatcher, "case_phases/", "delete")
}

func routeDeleteCasesByIdDelete(h httpevent.Event) uint32 {
	return grDelete(h, "cases")
}

func routeDeleteCasesByIdFiles(h httpevent.Event) uint32 {
	return grDelete(h, "cases")
}

func routeDeleteConsultationPackagesById(h httpevent.Event) uint32 {
	return handleDeleteConsultationPackage(h)
}

func routeDeleteFeedsClientCasesById(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_client_cases")
}

func routeDeleteFeedsClientConsultationsById(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_client_consultations")
}

func routeDeleteFeedsClientConsultationsByIdAnswersByAnswerId(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_client_consultations")
}

func routeDeleteFeedsLawyerConsultationsAnswersById(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_lawyer_consultations")
}

func routeDeleteFeedsLawyerConsultationsCommentsByCommentId(h httpevent.Event) uint32 {
	return grDelete(h, "feeds_lawyer_consultations")
}

func routeDeleteFoldersById(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersByIdShare(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersByIdShareAll(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersFilesById(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersFilesByIdShare(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteFoldersFilesByIdShareAll(h httpevent.Event) uint32 {
	return grDelete(h, "folders")
}

func routeDeleteForumPostsById(h httpevent.Event) uint32 {
	return grDelete(h, "forum_posts")
}

func routeDeleteForumRepliesById(h httpevent.Event) uint32 {
	return grDelete(h, "forum_replies")
}

func routeDeleteLawyerPackagesById(h httpevent.Event) uint32 {
	return handleDeleteLawyerPackage(h)
}

func routeDeletePetitionsById(h httpevent.Event) uint32 {
	return grDelete(h, "petitions")
}

func routeDeleteRequestsById(h httpevent.Event) uint32 {
	return grDelete(h, "requests")
}

func routeDeleteSpecializationsDeleteById(h httpevent.Event) uint32 {
	return localizedDelete(h, specializationsMatcher, "specializations/", "delete")
}

func routeDeleteStatesDeleteById(h httpevent.Event) uint32 {
	return handleDeleteStatesDeleteById(h)
}

func routeDeleteUserConsultationSubscriptionsById(h httpevent.Event) uint32 {
	return grDelete(h, "user_consultation_subscriptions")
}

func routeGetAdminReports(h httpevent.Event) uint32 {
	return handleGetAdminReportsNest(h)
}

func routeGetAssistants(h httpevent.Event) uint32 {
	return grList(h, "assistants")
}

func routeGetAssistantsByIdPermissions(h httpevent.Event) uint32 {
	return grList(h, "assistants")
}

func routeGetBanners(h httpevent.Event) uint32 {
	return handleGetBannersPublic(h, false)
}

func routeGetBannersActiveByType(h httpevent.Event) uint32 {
	return handleGetBannersPublic(h, true)
}

func routeGetBannersById(h httpevent.Event) uint32 {
	return handleBannerByID(h)
}

func routeGetCaseCategoriesGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, caseCategoriesMatcher, "case_categories/")
}

func routeGetCaseCategoriesIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, caseCategoriesMatcher, "case_categories/", "")
}

func routeGetCaseChambersGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, caseChambersMatcher, "case_chambers/")
}

func routeGetCaseChambersIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, caseChambersMatcher, "case_chambers/", "")
}

func routeGetCasePhasesGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, casePhasesMatcher, "case_phases/")
}

func routeGetCasePhasesIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, casePhasesMatcher, "case_phases/", "")
}

func routeGetCasesByIdDownload(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/cases/download"})
}

func routeGetCasesByIdGetCaseFiles(h httpevent.Event) uint32 {
	return grGetByID(h, "cases")
}

func routeGetCasesByIdGetCaseNotes(h httpevent.Event) uint32 {
	return grGetByID(h, "cases")
}

func routeGetCasesByIdUsersPermissions(h httpevent.Event) uint32 {
	return grGetByID(h, "cases")
}

func routeGetCasesGet(h httpevent.Event) uint32 {
	return grList(h, "cases")
}

func routeGetConsultationPackages(h httpevent.Event) uint32 {
	return handleListConsultationPackagesNest(h)
}

func routeGetConsultationPackagesById(h httpevent.Event) uint32 {
	return handleConsultationPackageByID(h)
}

func routeGetFeedsClientCases(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientCasesApplicationsAccepted(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientCasesById(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientCasesByIdApplications(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_cases")
}

func routeGetFeedsClientConsultations(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_consultations")
}

func routeGetFeedsClientConsultationsAnswers(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_consultations")
}

func routeGetFeedsClientConsultationsByIdAnswers(h httpevent.Event) uint32 {
	return grList(h, "feeds_client_consultations")
}

func routeGetFeedsLawyerCases(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_cases")
}

func routeGetFeedsLawyerCasesLawyerCases(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_cases")
}

func routeGetFeedsLawyerConsultations(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsAnswersMe(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsById(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsByIdAnswers(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFeedsLawyerConsultationsCommentsByAnswerId(h httpevent.Event) uint32 {
	return grList(h, "feeds_lawyer_consultations")
}

func routeGetFolders(h httpevent.Event) uint32 {
	return grList(h, "folders")
}

func routeGetFoldersById(h httpevent.Event) uint32 {
	return grList(h, "folders")
}

func routeGetFoldersByIdDownloadFolder(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/folders/dl"})
}

func routeGetFoldersByIdShare(h httpevent.Event) uint32 {
	return grList(h, "folders")
}

func routeGetFoldersFilesByIdDownload(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/folders/file/dl"})
}

func routeGetFoldersFilesByIdShare(h httpevent.Event) uint32 {
	return grList(h, "folder_file_shares")
}

func routeGetFoldersSharesReceived(h httpevent.Event) uint32 {
	return grList(h, "folder_shares")
}

func routeGetFoldersSharesReceivedUnreadCount(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"count": 0})
}

func routeGetFoldersSharesSent(h httpevent.Event) uint32 {
	return grList(h, "folder_shares")
}

func routeGetForumNotifications(h httpevent.Event) uint32 {
	return grList(h, "forum_notifications")
}

func routeGetForumPosts(h httpevent.Event) uint32 {
	return grList(h, "forum_posts")
}

func routeGetForumPostsById(h httpevent.Event) uint32 {
	return grGetByID(h, "forum_posts")
}

func routeGetForumPostsByIdReplies(h httpevent.Event) uint32 {
	return grList(h, "forum_replies")
}

func routeGetJudicalReqiests(h httpevent.Event) uint32 {
	return grList(h, "judicial_requests")
}

func routeGetJudicalReqiestsById(h httpevent.Event) uint32 {
	return grGetByID(h, "judicial_requests")
}

func routeGetLawyerPackages(h httpevent.Event) uint32 {
	return handleListLawyerPackagesNest(h)
}

func routeGetLawyerPackagesById(h httpevent.Event) uint32 {
	return handleLawyerPackageByID(h)
}

func routeGetLawyerPackagesSubscriptionsHistory(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetLawyerSubscriptionsAvailablePackages(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetLawyerSubscriptionsHistory(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetLawyerSubscriptionsStatus(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routeGetPermissions(h httpevent.Event) uint32 {
	return grList(h, "assistants")
}

func routeGetPetitionsById(h httpevent.Event) uint32 {
	return grGetByID(h, "petitions")
}

func routeGetPetitionsByIdPdf(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"url": "/petitions/pdf/" + pathLast(h)})
}

func routeGetPetitionsFinal(h httpevent.Event) uint32 {
	return grList(h, "petitions")
}

func routeGetRequests(h httpevent.Event) uint32 {
	return grList(h, "requests")
}

func routeGetRequestsMyRequests(h httpevent.Event) uint32 {
	return grList(h, "requests")
}

func routeGetRequestsOfficerAccepted(h httpevent.Event) uint32 {
	return grList(h, "requests")
}

func routeGetSpecializationsGet(h httpevent.Event) uint32 {
	return localizedGetPublic(h, specializationsMatcher, "specializations/")
}

func routeGetSpecializationsIndex(h httpevent.Event) uint32 {
	return localizedIndex(h, specializationsMatcher, "specializations/", "")
}

func routeGetStatesGet(h httpevent.Event) uint32 {
	return handleGetStatesGet(h)
}

func routeGetStatesIndex(h httpevent.Event) uint32 {
	return handleGetStatesIndex(h)
}

func routeGetUserConsultationSubscriptions(h httpevent.Event) uint32 {
	return grList(h, "user_consultation_subscriptions")
}

func routeGetUserConsultationSubscriptionsById(h httpevent.Event) uint32 {
	return grGetByID(h, "user_consultation_subscriptions")
}

func routeGetUsersJudicialOfficers(h httpevent.Event) uint32 {
	return handleGetUsersJudicialOfficers(h)
}

func routeGetUsersJudicialOfficersById(h httpevent.Event) uint32 {
	return handleGetUsersJudicialOfficersById(h)
}

func routeGetUsersLawyers(h httpevent.Event) uint32 {
	return handleGetUsersLawyers(h)
}

func routeGetUsersLawyersById(h httpevent.Event) uint32 {
	return handleGetUsersLawyersById(h)
}

func routeGetUsersLawyersDashboardStats(h httpevent.Event) uint32 {
	return handleGetUsersLawyersDashboardStats(h)
}

func routeGetUsersLawyersGet(h httpevent.Event) uint32 {
	return handleGetUsersLawyersGet(h)
}

func routeGetUsersLawyersLawyers(h httpevent.Event) uint32 {
	return handleGetUsersLawyersLawyers(h)
}

func routeGetUsersLawyersLawyersVerifiying(h httpevent.Event) uint32 {
	return handleGetUsersLawyersLawyersVerifiying(h)
}

func routeGetUsersMe(h httpevent.Event) uint32 {
	return handleGetUsersMe(h)
}

func routeGetUsersSearch(h httpevent.Event) uint32 {
	return handleGetUsersSearch(h)
}

func routePatchAssistantsByIdPermissions(h httpevent.Event) uint32 {
	return grPatchJSON(h, "assistants")
}

func routePatchBannersById(h httpevent.Event) uint32 {
	return handlePatchBanner(h)
}

func routePatchCaseCategoriesUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, caseCategoriesMatcher, "case_categories/", "update")
}

func routePatchCaseChambersUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, caseChambersMatcher, "case_chambers/", "update")
}

func routePatchCasePhasesUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, casePhasesMatcher, "case_phases/", "update")
}

func routePatchCasesByIdUpdate(h httpevent.Event) uint32 {
	return grPatchJSON(h, "cases")
}

func routePatchConsultationPackagesById(h httpevent.Event) uint32 {
	return handlePatchConsultationPackage(h)
}

func routePatchFeedsClientCasesById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_client_cases")
}

func routePatchFeedsClientConsultationsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_client_consultations")
}

func routePatchFeedsLawyerConsultationsAnswersById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_lawyer_consultations")
}

func routePatchFeedsLawyerConsultationsCommentsByCommentId(h httpevent.Event) uint32 {
	return grPatchJSON(h, "feeds_lawyer_consultations")
}

func routePatchFoldersById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "folders")
}

func routePatchFoldersFilesByIdRename(h httpevent.Event) uint32 {
	return grPatchJSON(h, "folders")
}

func routePatchForumPostsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_posts")
}

func routePatchLawyerPackagesById(h httpevent.Event) uint32 {
	return handlePatchLawyerPackage(h)
}

func routePatchRequestsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "requests")
}

func routePatchSpecializationsUpdateById(h httpevent.Event) uint32 {
	return localizedPatch(h, specializationsMatcher, "specializations/", "update")
}

func routePatchStatesUpdateById(h httpevent.Event) uint32 {
	return handlePatchStatesUpdateById(h)
}

func routePatchUserConsultationSubscriptionsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "user_consultation_subscriptions")
}

func routePatchUsersChangePassword(h httpevent.Event) uint32 {
	return handlePatchUsersChangePassword(h)
}

func routePatchUsersLawyersAcceptVerifiyingById(h httpevent.Event) uint32 {
	return handlePatchUsersLawyersAcceptVerifiyingById(h)
}

func routePatchUsersMe(h httpevent.Event) uint32 {
	return handlePatchUsersMe(h)
}

func routePostAdminReportsAction(h httpevent.Event) uint32 {
	return handlePostAdminReportsActionNest(h)
}

func routePostAssistants(h httpevent.Event) uint32 {
	return grPostJSON(h, "assistants")
}

func routePostAuthLogin(h httpevent.Event) uint32 {
	return handlePostAuthLogin(h)
}

func routePostAuthLoginGoogle(h httpevent.Event) uint32 {
	return handlePostAuthLoginGoogle(h)
}

func routePostAuthLogout(h httpevent.Event) uint32 {
	return handlePostAuthLogout(h)
}

func routePostAuthRefreshToken(h httpevent.Event) uint32 {
	return handlePostAuthRefreshToken(h)
}

func routePostAuthRegister(h httpevent.Event) uint32 {
	return handlePostAuthRegister(h)
}

func routePostAuthResendResetCode(h httpevent.Event) uint32 {
	return handlePostAuthResendResetCode(h)
}

func routePostAuthResendVerificationCode(h httpevent.Event) uint32 {
	return handlePostAuthResendVerificationCode(h)
}

func routePostAuthResetPassword(h httpevent.Event) uint32 {
	return handlePostAuthResetPassword(h)
}

func routePostAuthSetRole(h httpevent.Event) uint32 {
	return handlePostAuthSetRole(h)
}

func routePostAuthVerifyEmail(h httpevent.Event) uint32 {
	return handlePostAuthVerifyEmail(h)
}

func routePostAuthVerifyResetCode(h httpevent.Event) uint32 {
	return handlePostAuthVerifyResetCode(h)
}

func routePostBanners(h httpevent.Event) uint32 {
	return handlePostBanner(h)
}

func routePostBotChat(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"reply": "ok"})
}

func routePostCaseCategoriesStore(h httpevent.Event) uint32 {
	return localizedStore(h, caseCategoriesMatcher, "case_categories/", "category")
}

func routePostCaseChambersStore(h httpevent.Event) uint32 {
	return localizedStore(h, caseChambersMatcher, "case_chambers/", "chamber")
}

func routePostCasePhasesStore(h httpevent.Event) uint32 {
	return localizedStore(h, casePhasesMatcher, "case_phases/", "phase")
}

func routePostCasesByIdFileTitle(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdFiles(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdNotes(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdSaveCase(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesByIdShare(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostCasesStore(h httpevent.Event) uint32 {
	return grPostJSON(h, "cases")
}

func routePostConsultationPackages(h httpevent.Event) uint32 {
	return handlePostConsultationPackage(h)
}

func routePostFeedsClientCases(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_cases")
}

func routePostFeedsClientCasesByIdApplicationsAcceptByApplicationId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_cases")
}

func routePostFeedsClientCasesByIdApplicationsRejectByApplicationId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_cases")
}

func routePostFeedsClientConsultations(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_consultations")
}

func routePostFeedsClientConsultationsByIdAnswersByAnswerId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_client_consultations")
}

func routePostFeedsLawyerCasesByIdApply(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_lawyer_cases")
}

func routePostFeedsLawyerConsultationsByIdAnswers(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_lawyer_consultations")
}

func routePostFeedsLawyerConsultationsCommentsByAnswerId(h httpevent.Event) uint32 {
	return grPostJSON(h, "feeds_lawyer_consultations")
}

func routePostFolders(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostFoldersByIdFiles(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostFoldersByIdShare(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostFoldersFilesByIdShare(h httpevent.Event) uint32 {
	return grPostJSON(h, "folders")
}

func routePostForumPosts(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_posts")
}

func routePostForumPostsHide(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_posts")
}

func routePostForumPostsReport(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_posts")
}

func routePostForumReplies(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_replies")
}

func routePostForumRepliesReport(h httpevent.Event) uint32 {
	return grPostJSON(h, "forum_replies")
}

func routePostLawyerPackages(h httpevent.Event) uint32 {
	return handlePostLawyerPackage(h)
}

func routePostLawyerPackagesSubscriptionsCleanup(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routePostLawyerSubscriptionsSubscribe(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"status": "ok"})
}

func routePostPetitions(h httpevent.Event) uint32 {
	return grPostJSON(h, "petitions")
}

func routePostPetitionsByIdMove(h httpevent.Event) uint32 {
	return grPatchJSON(h, "petitions")
}

func routePostPetitionsImage(h httpevent.Event) uint32 {
	return grPostJSON(h, "petitions")
}

func routePostPetitionsUploadFile(h httpevent.Event) uint32 {
	return grPostJSON(h, "petitions")
}

func routePostRequests(h httpevent.Event) uint32 {
	return grPostJSON(h, "requests")
}

func routePostRequestsByIdUpdateStatus(h httpevent.Event) uint32 {
	return grPatchJSON(h, "requests")
}

func routePostSpecializationsStore(h httpevent.Event) uint32 {
	return localizedStore(h, specializationsMatcher, "specializations/", "spec")
}

func routePostStatesStore(h httpevent.Event) uint32 {
	return handlePostStatesStore(h)
}

func routePostUserConsultationSubscriptions(h httpevent.Event) uint32 {
	return grPostJSON(h, "user_consultation_subscriptions")
}

func routePostUsersChatNotification(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"sent": true})
}

func routePostUsersCreateImage(h httpevent.Event) uint32 {
	return handlePostUsersCreateImage(h)
}

func routePostUsersLawyerVerificationFiles(h httpevent.Event) uint32 {
	return grOKMap(h, map[string]any{"uploaded": true})
}

func routePostUsersRemoveImage(h httpevent.Event) uint32 {
	return handlePostUsersRemoveImage(h)
}

func routePostUsersRemoveImageCover(h httpevent.Event) uint32 {
	return handlePostUsersRemoveImageCover(h)
}

func routePostUsersSavedAiImage(h httpevent.Event) uint32 {
	return handlePostUsersSavedAiImage(h)
}

func routePostUsersUploadImage(h httpevent.Event) uint32 {
	return handlePostUsersUploadImage(h)
}

func routePostUsersUploadImageCover(h httpevent.Event) uint32 {
	return handlePostUsersUploadImageCover(h)
}

func routePutCasesByIdRelations(h httpevent.Event) uint32 {
	return grPatchJSON(h, "cases")
}

func routePutForumNotificationsByIdRead(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_notifications")
}

func routePutForumRepliesById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "forum_replies")
}

func routePutPetitionsById(h httpevent.Event) uint32 {
	return grPatchJSON(h, "petitions")
}

func routePutUsersLawyersAcceptByUserId(h httpevent.Event) uint32 {
	return handlePutUsersLawyersAcceptByUserId(h)
}

func routePutUsersLawyersRejectByUserId(h httpevent.Event) uint32 {
	return handlePutUsersLawyersRejectByUserId(h)
}

// Code generated by project agent — dispatch table for lawgen_backend HTTP functions.

func dispatchLawgenHandler(name string, h httpevent.Event) uint32 {
	switch name {
	case "DeleteAssistants":
		return routeDeleteAssistants(h)
	case "DeleteBannersById":
		return routeDeleteBannersById(h)
	case "DeleteCaseCategoriesDeleteById":
		return routeDeleteCaseCategoriesDeleteById(h)
	case "DeleteCaseChambersDeleteById":
		return routeDeleteCaseChambersDeleteById(h)
	case "DeleteCasePhasesDeleteById":
		return routeDeleteCasePhasesDeleteById(h)
	case "DeleteCasesByIdDelete":
		return routeDeleteCasesByIdDelete(h)
	case "DeleteCasesByIdFiles":
		return routeDeleteCasesByIdFiles(h)
	case "DeleteConsultationPackagesById":
		return routeDeleteConsultationPackagesById(h)
	case "DeleteFeedsClientCasesById":
		return routeDeleteFeedsClientCasesById(h)
	case "DeleteFeedsClientConsultationsById":
		return routeDeleteFeedsClientConsultationsById(h)
	case "DeleteFeedsClientConsultationsByIdAnswersByAnswerId":
		return routeDeleteFeedsClientConsultationsByIdAnswersByAnswerId(h)
	case "DeleteFeedsLawyerConsultationsAnswersById":
		return routeDeleteFeedsLawyerConsultationsAnswersById(h)
	case "DeleteFeedsLawyerConsultationsCommentsByCommentId":
		return routeDeleteFeedsLawyerConsultationsCommentsByCommentId(h)
	case "DeleteFoldersById":
		return routeDeleteFoldersById(h)
	case "DeleteFoldersByIdShare":
		return routeDeleteFoldersByIdShare(h)
	case "DeleteFoldersByIdShareAll":
		return routeDeleteFoldersByIdShareAll(h)
	case "DeleteFoldersFilesById":
		return routeDeleteFoldersFilesById(h)
	case "DeleteFoldersFilesByIdShare":
		return routeDeleteFoldersFilesByIdShare(h)
	case "DeleteFoldersFilesByIdShareAll":
		return routeDeleteFoldersFilesByIdShareAll(h)
	case "DeleteForumPostsById":
		return routeDeleteForumPostsById(h)
	case "DeleteForumRepliesById":
		return routeDeleteForumRepliesById(h)
	case "DeleteLawyerPackagesById":
		return routeDeleteLawyerPackagesById(h)
	case "DeletePetitionsById":
		return routeDeletePetitionsById(h)
	case "DeleteRequestsById":
		return routeDeleteRequestsById(h)
	case "DeleteSpecializationsDeleteById":
		return routeDeleteSpecializationsDeleteById(h)
	case "DeleteStatesDeleteById":
		return routeDeleteStatesDeleteById(h)
	case "DeleteUserConsultationSubscriptionsById":
		return routeDeleteUserConsultationSubscriptionsById(h)
	case "GetAdminReports":
		return routeGetAdminReports(h)
	case "GetAssistants":
		return routeGetAssistants(h)
	case "GetAssistantsByIdPermissions":
		return routeGetAssistantsByIdPermissions(h)
	case "GetBanners":
		return routeGetBanners(h)
	case "GetBannersActiveByType":
		return routeGetBannersActiveByType(h)
	case "GetBannersById":
		return routeGetBannersById(h)
	case "GetCaseCategoriesGet":
		return routeGetCaseCategoriesGet(h)
	case "GetCaseCategoriesIndex":
		return routeGetCaseCategoriesIndex(h)
	case "GetCaseChambersGet":
		return routeGetCaseChambersGet(h)
	case "GetCaseChambersIndex":
		return routeGetCaseChambersIndex(h)
	case "GetCasePhasesGet":
		return routeGetCasePhasesGet(h)
	case "GetCasePhasesIndex":
		return routeGetCasePhasesIndex(h)
	case "GetCasesByIdDownload":
		return routeGetCasesByIdDownload(h)
	case "GetCasesByIdGetCaseFiles":
		return routeGetCasesByIdGetCaseFiles(h)
	case "GetCasesByIdGetCaseNotes":
		return routeGetCasesByIdGetCaseNotes(h)
	case "GetCasesByIdUsersPermissions":
		return routeGetCasesByIdUsersPermissions(h)
	case "GetCasesGet":
		return routeGetCasesGet(h)
	case "GetConsultationPackages":
		return routeGetConsultationPackages(h)
	case "GetConsultationPackagesById":
		return routeGetConsultationPackagesById(h)
	case "GetFeedsClientCases":
		return routeGetFeedsClientCases(h)
	case "GetFeedsClientCasesApplicationsAccepted":
		return routeGetFeedsClientCasesApplicationsAccepted(h)
	case "GetFeedsClientCasesById":
		return routeGetFeedsClientCasesById(h)
	case "GetFeedsClientCasesByIdApplications":
		return routeGetFeedsClientCasesByIdApplications(h)
	case "GetFeedsClientConsultations":
		return routeGetFeedsClientConsultations(h)
	case "GetFeedsClientConsultationsAnswers":
		return routeGetFeedsClientConsultationsAnswers(h)
	case "GetFeedsClientConsultationsByIdAnswers":
		return routeGetFeedsClientConsultationsByIdAnswers(h)
	case "GetFeedsLawyerCases":
		return routeGetFeedsLawyerCases(h)
	case "GetFeedsLawyerCasesLawyerCases":
		return routeGetFeedsLawyerCasesLawyerCases(h)
	case "GetFeedsLawyerConsultations":
		return routeGetFeedsLawyerConsultations(h)
	case "GetFeedsLawyerConsultationsAnswersMe":
		return routeGetFeedsLawyerConsultationsAnswersMe(h)
	case "GetFeedsLawyerConsultationsById":
		return routeGetFeedsLawyerConsultationsById(h)
	case "GetFeedsLawyerConsultationsByIdAnswers":
		return routeGetFeedsLawyerConsultationsByIdAnswers(h)
	case "GetFeedsLawyerConsultationsCommentsByAnswerId":
		return routeGetFeedsLawyerConsultationsCommentsByAnswerId(h)
	case "GetFolders":
		return routeGetFolders(h)
	case "GetFoldersById":
		return routeGetFoldersById(h)
	case "GetFoldersByIdDownloadFolder":
		return routeGetFoldersByIdDownloadFolder(h)
	case "GetFoldersByIdShare":
		return routeGetFoldersByIdShare(h)
	case "GetFoldersFilesByIdDownload":
		return routeGetFoldersFilesByIdDownload(h)
	case "GetFoldersFilesByIdShare":
		return routeGetFoldersFilesByIdShare(h)
	case "GetFoldersSharesReceived":
		return routeGetFoldersSharesReceived(h)
	case "GetFoldersSharesReceivedUnreadCount":
		return routeGetFoldersSharesReceivedUnreadCount(h)
	case "GetFoldersSharesSent":
		return routeGetFoldersSharesSent(h)
	case "GetForumNotifications":
		return routeGetForumNotifications(h)
	case "GetForumPosts":
		return routeGetForumPosts(h)
	case "GetForumPostsById":
		return routeGetForumPostsById(h)
	case "GetForumPostsByIdReplies":
		return routeGetForumPostsByIdReplies(h)
	case "GetJudicalReqiests":
		return routeGetJudicalReqiests(h)
	case "GetJudicalReqiestsById":
		return routeGetJudicalReqiestsById(h)
	case "GetLawyerPackages":
		return routeGetLawyerPackages(h)
	case "GetLawyerPackagesById":
		return routeGetLawyerPackagesById(h)
	case "GetLawyerPackagesSubscriptionsHistory":
		return routeGetLawyerPackagesSubscriptionsHistory(h)
	case "GetLawyerSubscriptionsAvailablePackages":
		return routeGetLawyerSubscriptionsAvailablePackages(h)
	case "GetLawyerSubscriptionsHistory":
		return routeGetLawyerSubscriptionsHistory(h)
	case "GetLawyerSubscriptionsStatus":
		return routeGetLawyerSubscriptionsStatus(h)
	case "GetPermissions":
		return routeGetPermissions(h)
	case "GetPetitionsById":
		return routeGetPetitionsById(h)
	case "GetPetitionsByIdPdf":
		return routeGetPetitionsByIdPdf(h)
	case "GetPetitionsFinal":
		return routeGetPetitionsFinal(h)
	case "GetRequests":
		return routeGetRequests(h)
	case "GetRequestsMyRequests":
		return routeGetRequestsMyRequests(h)
	case "GetRequestsOfficerAccepted":
		return routeGetRequestsOfficerAccepted(h)
	case "GetSpecializationsGet":
		return routeGetSpecializationsGet(h)
	case "GetSpecializationsIndex":
		return routeGetSpecializationsIndex(h)
	case "GetStatesGet":
		return routeGetStatesGet(h)
	case "GetStatesIndex":
		return routeGetStatesIndex(h)
	case "GetUserConsultationSubscriptions":
		return routeGetUserConsultationSubscriptions(h)
	case "GetUserConsultationSubscriptionsById":
		return routeGetUserConsultationSubscriptionsById(h)
	case "GetUsersJudicialOfficers":
		return routeGetUsersJudicialOfficers(h)
	case "GetUsersJudicialOfficersById":
		return routeGetUsersJudicialOfficersById(h)
	case "GetUsersLawyers":
		return routeGetUsersLawyers(h)
	case "GetUsersLawyersById":
		return routeGetUsersLawyersById(h)
	case "GetUsersLawyersDashboardStats":
		return routeGetUsersLawyersDashboardStats(h)
	case "GetUsersLawyersGet":
		return routeGetUsersLawyersGet(h)
	case "GetUsersLawyersLawyers":
		return routeGetUsersLawyersLawyers(h)
	case "GetUsersLawyersLawyersVerifiying":
		return routeGetUsersLawyersLawyersVerifiying(h)
	case "GetUsersMe":
		return routeGetUsersMe(h)
	case "GetUsersSearch":
		return routeGetUsersSearch(h)
	case "PatchAssistantsByIdPermissions":
		return routePatchAssistantsByIdPermissions(h)
	case "PatchBannersById":
		return routePatchBannersById(h)
	case "PatchCaseCategoriesUpdateById":
		return routePatchCaseCategoriesUpdateById(h)
	case "PatchCaseChambersUpdateById":
		return routePatchCaseChambersUpdateById(h)
	case "PatchCasePhasesUpdateById":
		return routePatchCasePhasesUpdateById(h)
	case "PatchCasesByIdUpdate":
		return routePatchCasesByIdUpdate(h)
	case "PatchConsultationPackagesById":
		return routePatchConsultationPackagesById(h)
	case "PatchFeedsClientCasesById":
		return routePatchFeedsClientCasesById(h)
	case "PatchFeedsClientConsultationsById":
		return routePatchFeedsClientConsultationsById(h)
	case "PatchFeedsLawyerConsultationsAnswersById":
		return routePatchFeedsLawyerConsultationsAnswersById(h)
	case "PatchFeedsLawyerConsultationsCommentsByCommentId":
		return routePatchFeedsLawyerConsultationsCommentsByCommentId(h)
	case "PatchFoldersById":
		return routePatchFoldersById(h)
	case "PatchFoldersFilesByIdRename":
		return routePatchFoldersFilesByIdRename(h)
	case "PatchForumPostsById":
		return routePatchForumPostsById(h)
	case "PatchLawyerPackagesById":
		return routePatchLawyerPackagesById(h)
	case "PatchRequestsById":
		return routePatchRequestsById(h)
	case "PatchSpecializationsUpdateById":
		return routePatchSpecializationsUpdateById(h)
	case "PatchStatesUpdateById":
		return routePatchStatesUpdateById(h)
	case "PatchUserConsultationSubscriptionsById":
		return routePatchUserConsultationSubscriptionsById(h)
	case "PatchUsersChangePassword":
		return routePatchUsersChangePassword(h)
	case "PatchUsersLawyersAcceptVerifiyingById":
		return routePatchUsersLawyersAcceptVerifiyingById(h)
	case "PatchUsersMe":
		return routePatchUsersMe(h)
	case "PostAdminReportsAction":
		return routePostAdminReportsAction(h)
	case "PostAssistants":
		return routePostAssistants(h)
	case "PostAuthLogin":
		return routePostAuthLogin(h)
	case "PostAuthLoginGoogle":
		return routePostAuthLoginGoogle(h)
	case "PostAuthLogout":
		return routePostAuthLogout(h)
	case "PostAuthRefreshToken":
		return routePostAuthRefreshToken(h)
	case "PostAuthRegister":
		return routePostAuthRegister(h)
	case "PostAuthResendResetCode":
		return routePostAuthResendResetCode(h)
	case "PostAuthResendVerificationCode":
		return routePostAuthResendVerificationCode(h)
	case "PostAuthResetPassword":
		return routePostAuthResetPassword(h)
	case "PostAuthSetRole":
		return routePostAuthSetRole(h)
	case "PostAuthVerifyEmail":
		return routePostAuthVerifyEmail(h)
	case "PostAuthVerifyResetCode":
		return routePostAuthVerifyResetCode(h)
	case "PostBanners":
		return routePostBanners(h)
	case "PostBotChat":
		return routePostBotChat(h)
	case "PostCaseCategoriesStore":
		return routePostCaseCategoriesStore(h)
	case "PostCaseChambersStore":
		return routePostCaseChambersStore(h)
	case "PostCasePhasesStore":
		return routePostCasePhasesStore(h)
	case "PostCasesByIdFileTitle":
		return routePostCasesByIdFileTitle(h)
	case "PostCasesByIdFiles":
		return routePostCasesByIdFiles(h)
	case "PostCasesByIdNotes":
		return routePostCasesByIdNotes(h)
	case "PostCasesByIdSaveCase":
		return routePostCasesByIdSaveCase(h)
	case "PostCasesByIdShare":
		return routePostCasesByIdShare(h)
	case "PostCasesStore":
		return routePostCasesStore(h)
	case "PostConsultationPackages":
		return routePostConsultationPackages(h)
	case "PostFeedsClientCases":
		return routePostFeedsClientCases(h)
	case "PostFeedsClientCasesByIdApplicationsAcceptByApplicationId":
		return routePostFeedsClientCasesByIdApplicationsAcceptByApplicationId(h)
	case "PostFeedsClientCasesByIdApplicationsRejectByApplicationId":
		return routePostFeedsClientCasesByIdApplicationsRejectByApplicationId(h)
	case "PostFeedsClientConsultations":
		return routePostFeedsClientConsultations(h)
	case "PostFeedsClientConsultationsByIdAnswersByAnswerId":
		return routePostFeedsClientConsultationsByIdAnswersByAnswerId(h)
	case "PostFeedsLawyerCasesByIdApply":
		return routePostFeedsLawyerCasesByIdApply(h)
	case "PostFeedsLawyerConsultationsByIdAnswers":
		return routePostFeedsLawyerConsultationsByIdAnswers(h)
	case "PostFeedsLawyerConsultationsCommentsByAnswerId":
		return routePostFeedsLawyerConsultationsCommentsByAnswerId(h)
	case "PostFolders":
		return routePostFolders(h)
	case "PostFoldersByIdFiles":
		return routePostFoldersByIdFiles(h)
	case "PostFoldersByIdShare":
		return routePostFoldersByIdShare(h)
	case "PostFoldersFilesByIdShare":
		return routePostFoldersFilesByIdShare(h)
	case "PostForumPosts":
		return routePostForumPosts(h)
	case "PostForumPostsHide":
		return routePostForumPostsHide(h)
	case "PostForumPostsReport":
		return routePostForumPostsReport(h)
	case "PostForumReplies":
		return routePostForumReplies(h)
	case "PostForumRepliesReport":
		return routePostForumRepliesReport(h)
	case "PostLawyerPackages":
		return routePostLawyerPackages(h)
	case "PostLawyerPackagesSubscriptionsCleanup":
		return routePostLawyerPackagesSubscriptionsCleanup(h)
	case "PostLawyerSubscriptionsSubscribe":
		return routePostLawyerSubscriptionsSubscribe(h)
	case "PostPetitions":
		return routePostPetitions(h)
	case "PostPetitionsByIdMove":
		return routePostPetitionsByIdMove(h)
	case "PostPetitionsImage":
		return routePostPetitionsImage(h)
	case "PostPetitionsUploadFile":
		return routePostPetitionsUploadFile(h)
	case "PostRequests":
		return routePostRequests(h)
	case "PostRequestsByIdUpdateStatus":
		return routePostRequestsByIdUpdateStatus(h)
	case "PostSpecializationsStore":
		return routePostSpecializationsStore(h)
	case "PostStatesStore":
		return routePostStatesStore(h)
	case "PostUserConsultationSubscriptions":
		return routePostUserConsultationSubscriptions(h)
	case "PostUsersChatNotification":
		return routePostUsersChatNotification(h)
	case "PostUsersCreateImage":
		return routePostUsersCreateImage(h)
	case "PostUsersLawyerVerificationFiles":
		return routePostUsersLawyerVerificationFiles(h)
	case "PostUsersRemoveImage":
		return routePostUsersRemoveImage(h)
	case "PostUsersRemoveImageCover":
		return routePostUsersRemoveImageCover(h)
	case "PostUsersSavedAiImage":
		return routePostUsersSavedAiImage(h)
	case "PostUsersUploadImage":
		return routePostUsersUploadImage(h)
	case "PostUsersUploadImageCover":
		return routePostUsersUploadImageCover(h)
	case "PutCasesByIdRelations":
		return routePutCasesByIdRelations(h)
	case "PutForumNotificationsByIdRead":
		return routePutForumNotificationsByIdRead(h)
	case "PutForumRepliesById":
		return routePutForumRepliesById(h)
	case "PutPetitionsById":
		return routePutPetitionsById(h)
	case "PutUsersLawyersAcceptByUserId":
		return routePutUsersLawyersAcceptByUserId(h)
	case "PutUsersLawyersRejectByUserId":
		return routePutUsersLawyersRejectByUserId(h)
	default:
		return writeNestError(h, 501, "not implemented")
	}
}

// Code generated — WASM export wrappers.

//export DeleteAssistants
func DeleteAssistants(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteAssistants", h)
}

//export DeleteBannersById
func DeleteBannersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteBannersById", h)
}

//export DeleteCaseCategoriesDeleteById
func DeleteCaseCategoriesDeleteById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteCaseCategoriesDeleteById", h)
}

//export DeleteCaseChambersDeleteById
func DeleteCaseChambersDeleteById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteCaseChambersDeleteById", h)
}

//export DeleteCasePhasesDeleteById
func DeleteCasePhasesDeleteById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteCasePhasesDeleteById", h)
}

//export DeleteCasesByIdDelete
func DeleteCasesByIdDelete(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteCasesByIdDelete", h)
}

//export DeleteCasesByIdFiles
func DeleteCasesByIdFiles(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteCasesByIdFiles", h)
}

//export DeleteConsultationPackagesById
func DeleteConsultationPackagesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteConsultationPackagesById", h)
}

//export DeleteFeedsClientCasesById
func DeleteFeedsClientCasesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFeedsClientCasesById", h)
}

//export DeleteFeedsClientConsultationsById
func DeleteFeedsClientConsultationsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFeedsClientConsultationsById", h)
}

//export DeleteFeedsClientConsultationsByIdAnswersByAnswerId
func DeleteFeedsClientConsultationsByIdAnswersByAnswerId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFeedsClientConsultationsByIdAnswersByAnswerId", h)
}

//export DeleteFeedsLawyerConsultationsAnswersById
func DeleteFeedsLawyerConsultationsAnswersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFeedsLawyerConsultationsAnswersById", h)
}

//export DeleteFeedsLawyerConsultationsCommentsByCommentId
func DeleteFeedsLawyerConsultationsCommentsByCommentId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFeedsLawyerConsultationsCommentsByCommentId", h)
}

//export DeleteFoldersById
func DeleteFoldersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFoldersById", h)
}

//export DeleteFoldersByIdShare
func DeleteFoldersByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFoldersByIdShare", h)
}

//export DeleteFoldersByIdShareAll
func DeleteFoldersByIdShareAll(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFoldersByIdShareAll", h)
}

//export DeleteFoldersFilesById
func DeleteFoldersFilesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFoldersFilesById", h)
}

//export DeleteFoldersFilesByIdShare
func DeleteFoldersFilesByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFoldersFilesByIdShare", h)
}

//export DeleteFoldersFilesByIdShareAll
func DeleteFoldersFilesByIdShareAll(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteFoldersFilesByIdShareAll", h)
}

//export DeleteForumPostsById
func DeleteForumPostsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteForumPostsById", h)
}

//export DeleteForumRepliesById
func DeleteForumRepliesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteForumRepliesById", h)
}

//export DeleteLawyerPackagesById
func DeleteLawyerPackagesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteLawyerPackagesById", h)
}

//export DeletePetitionsById
func DeletePetitionsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeletePetitionsById", h)
}

//export DeleteRequestsById
func DeleteRequestsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteRequestsById", h)
}

//export DeleteSpecializationsDeleteById
func DeleteSpecializationsDeleteById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteSpecializationsDeleteById", h)
}

//export DeleteStatesDeleteById
func DeleteStatesDeleteById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteStatesDeleteById", h)
}

//export DeleteUserConsultationSubscriptionsById
func DeleteUserConsultationSubscriptionsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("DeleteUserConsultationSubscriptionsById", h)
}

//export GetAdminReports
func GetAdminReports(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetAdminReports", h)
}

//export GetAssistants
func GetAssistants(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetAssistants", h)
}

//export GetAssistantsByIdPermissions
func GetAssistantsByIdPermissions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetAssistantsByIdPermissions", h)
}

//export GetBanners
func GetBanners(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetBanners", h)
}

//export GetBannersActiveByType
func GetBannersActiveByType(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetBannersActiveByType", h)
}

//export GetBannersById
func GetBannersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetBannersById", h)
}

//export GetCaseCategoriesGet
func GetCaseCategoriesGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCaseCategoriesGet", h)
}

//export GetCaseCategoriesIndex
func GetCaseCategoriesIndex(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCaseCategoriesIndex", h)
}

//export GetCaseChambersGet
func GetCaseChambersGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCaseChambersGet", h)
}

//export GetCaseChambersIndex
func GetCaseChambersIndex(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCaseChambersIndex", h)
}

//export GetCasePhasesGet
func GetCasePhasesGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasePhasesGet", h)
}

//export GetCasePhasesIndex
func GetCasePhasesIndex(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasePhasesIndex", h)
}

//export GetCasesByIdDownload
func GetCasesByIdDownload(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasesByIdDownload", h)
}

//export GetCasesByIdGetCaseFiles
func GetCasesByIdGetCaseFiles(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasesByIdGetCaseFiles", h)
}

//export GetCasesByIdGetCaseNotes
func GetCasesByIdGetCaseNotes(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasesByIdGetCaseNotes", h)
}

//export GetCasesByIdUsersPermissions
func GetCasesByIdUsersPermissions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasesByIdUsersPermissions", h)
}

//export GetCasesGet
func GetCasesGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetCasesGet", h)
}

//export GetConsultationPackages
func GetConsultationPackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetConsultationPackages", h)
}

//export GetConsultationPackagesById
func GetConsultationPackagesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetConsultationPackagesById", h)
}

//export GetFeedsClientCases
func GetFeedsClientCases(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientCases", h)
}

//export GetFeedsClientCasesApplicationsAccepted
func GetFeedsClientCasesApplicationsAccepted(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientCasesApplicationsAccepted", h)
}

//export GetFeedsClientCasesById
func GetFeedsClientCasesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientCasesById", h)
}

//export GetFeedsClientCasesByIdApplications
func GetFeedsClientCasesByIdApplications(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientCasesByIdApplications", h)
}

//export GetFeedsClientConsultations
func GetFeedsClientConsultations(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientConsultations", h)
}

//export GetFeedsClientConsultationsAnswers
func GetFeedsClientConsultationsAnswers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientConsultationsAnswers", h)
}

//export GetFeedsClientConsultationsByIdAnswers
func GetFeedsClientConsultationsByIdAnswers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsClientConsultationsByIdAnswers", h)
}

//export GetFeedsLawyerCases
func GetFeedsLawyerCases(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerCases", h)
}

//export GetFeedsLawyerCasesLawyerCases
func GetFeedsLawyerCasesLawyerCases(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerCasesLawyerCases", h)
}

//export GetFeedsLawyerConsultations
func GetFeedsLawyerConsultations(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerConsultations", h)
}

//export GetFeedsLawyerConsultationsAnswersMe
func GetFeedsLawyerConsultationsAnswersMe(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerConsultationsAnswersMe", h)
}

//export GetFeedsLawyerConsultationsById
func GetFeedsLawyerConsultationsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerConsultationsById", h)
}

//export GetFeedsLawyerConsultationsByIdAnswers
func GetFeedsLawyerConsultationsByIdAnswers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerConsultationsByIdAnswers", h)
}

//export GetFeedsLawyerConsultationsCommentsByAnswerId
func GetFeedsLawyerConsultationsCommentsByAnswerId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFeedsLawyerConsultationsCommentsByAnswerId", h)
}

//export GetFolders
func GetFolders(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFolders", h)
}

//export GetFoldersById
func GetFoldersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersById", h)
}

//export GetFoldersByIdDownloadFolder
func GetFoldersByIdDownloadFolder(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersByIdDownloadFolder", h)
}

//export GetFoldersByIdShare
func GetFoldersByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersByIdShare", h)
}

//export GetFoldersFilesByIdDownload
func GetFoldersFilesByIdDownload(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersFilesByIdDownload", h)
}

//export GetFoldersFilesByIdShare
func GetFoldersFilesByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersFilesByIdShare", h)
}

//export GetFoldersSharesReceived
func GetFoldersSharesReceived(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersSharesReceived", h)
}

//export GetFoldersSharesReceivedUnreadCount
func GetFoldersSharesReceivedUnreadCount(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersSharesReceivedUnreadCount", h)
}

//export GetFoldersSharesSent
func GetFoldersSharesSent(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetFoldersSharesSent", h)
}

//export GetForumNotifications
func GetForumNotifications(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetForumNotifications", h)
}

//export GetForumPosts
func GetForumPosts(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetForumPosts", h)
}

//export GetForumPostsById
func GetForumPostsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetForumPostsById", h)
}

//export GetForumPostsByIdReplies
func GetForumPostsByIdReplies(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetForumPostsByIdReplies", h)
}

//export GetJudicalReqiests
func GetJudicalReqiests(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetJudicalReqiests", h)
}

//export GetJudicalReqiestsById
func GetJudicalReqiestsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetJudicalReqiestsById", h)
}

//export GetLawyerPackages
func GetLawyerPackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetLawyerPackages", h)
}

//export GetLawyerPackagesById
func GetLawyerPackagesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetLawyerPackagesById", h)
}

//export GetLawyerPackagesSubscriptionsHistory
func GetLawyerPackagesSubscriptionsHistory(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetLawyerPackagesSubscriptionsHistory", h)
}

//export GetLawyerSubscriptionsAvailablePackages
func GetLawyerSubscriptionsAvailablePackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetLawyerSubscriptionsAvailablePackages", h)
}

//export GetLawyerSubscriptionsHistory
func GetLawyerSubscriptionsHistory(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetLawyerSubscriptionsHistory", h)
}

//export GetLawyerSubscriptionsStatus
func GetLawyerSubscriptionsStatus(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetLawyerSubscriptionsStatus", h)
}

//export GetPermissions
func GetPermissions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetPermissions", h)
}

//export GetPetitionsById
func GetPetitionsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetPetitionsById", h)
}

//export GetPetitionsByIdPdf
func GetPetitionsByIdPdf(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetPetitionsByIdPdf", h)
}

//export GetPetitionsFinal
func GetPetitionsFinal(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetPetitionsFinal", h)
}

//export GetRequests
func GetRequests(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetRequests", h)
}

//export GetRequestsMyRequests
func GetRequestsMyRequests(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetRequestsMyRequests", h)
}

//export GetRequestsOfficerAccepted
func GetRequestsOfficerAccepted(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetRequestsOfficerAccepted", h)
}

//export GetSpecializationsGet
func GetSpecializationsGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetSpecializationsGet", h)
}

//export GetSpecializationsIndex
func GetSpecializationsIndex(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetSpecializationsIndex", h)
}

//export GetStatesGet
func GetStatesGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetStatesGet", h)
}

//export GetStatesIndex
func GetStatesIndex(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetStatesIndex", h)
}

//export GetUserConsultationSubscriptions
func GetUserConsultationSubscriptions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUserConsultationSubscriptions", h)
}

//export GetUserConsultationSubscriptionsById
func GetUserConsultationSubscriptionsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUserConsultationSubscriptionsById", h)
}

//export GetUsersJudicialOfficers
func GetUsersJudicialOfficers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersJudicialOfficers", h)
}

//export GetUsersJudicialOfficersById
func GetUsersJudicialOfficersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersJudicialOfficersById", h)
}

//export GetUsersLawyers
func GetUsersLawyers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersLawyers", h)
}

//export GetUsersLawyersById
func GetUsersLawyersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersLawyersById", h)
}

//export GetUsersLawyersDashboardStats
func GetUsersLawyersDashboardStats(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersLawyersDashboardStats", h)
}

//export GetUsersLawyersGet
func GetUsersLawyersGet(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersLawyersGet", h)
}

//export GetUsersLawyersLawyers
func GetUsersLawyersLawyers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersLawyersLawyers", h)
}

//export GetUsersLawyersLawyersVerifiying
func GetUsersLawyersLawyersVerifiying(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersLawyersLawyersVerifiying", h)
}

//export GetUsersMe
func GetUsersMe(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersMe", h)
}

//export GetUsersSearch
func GetUsersSearch(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("GetUsersSearch", h)
}

//export PatchAssistantsByIdPermissions
func PatchAssistantsByIdPermissions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchAssistantsByIdPermissions", h)
}

//export PatchBannersById
func PatchBannersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchBannersById", h)
}

//export PatchCaseCategoriesUpdateById
func PatchCaseCategoriesUpdateById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchCaseCategoriesUpdateById", h)
}

//export PatchCaseChambersUpdateById
func PatchCaseChambersUpdateById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchCaseChambersUpdateById", h)
}

//export PatchCasePhasesUpdateById
func PatchCasePhasesUpdateById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchCasePhasesUpdateById", h)
}

//export PatchCasesByIdUpdate
func PatchCasesByIdUpdate(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchCasesByIdUpdate", h)
}

//export PatchConsultationPackagesById
func PatchConsultationPackagesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchConsultationPackagesById", h)
}

//export PatchFeedsClientCasesById
func PatchFeedsClientCasesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchFeedsClientCasesById", h)
}

//export PatchFeedsClientConsultationsById
func PatchFeedsClientConsultationsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchFeedsClientConsultationsById", h)
}

//export PatchFeedsLawyerConsultationsAnswersById
func PatchFeedsLawyerConsultationsAnswersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchFeedsLawyerConsultationsAnswersById", h)
}

//export PatchFeedsLawyerConsultationsCommentsByCommentId
func PatchFeedsLawyerConsultationsCommentsByCommentId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchFeedsLawyerConsultationsCommentsByCommentId", h)
}

//export PatchFoldersById
func PatchFoldersById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchFoldersById", h)
}

//export PatchFoldersFilesByIdRename
func PatchFoldersFilesByIdRename(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchFoldersFilesByIdRename", h)
}

//export PatchForumPostsById
func PatchForumPostsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchForumPostsById", h)
}

//export PatchLawyerPackagesById
func PatchLawyerPackagesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchLawyerPackagesById", h)
}

//export PatchRequestsById
func PatchRequestsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchRequestsById", h)
}

//export PatchSpecializationsUpdateById
func PatchSpecializationsUpdateById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchSpecializationsUpdateById", h)
}

//export PatchStatesUpdateById
func PatchStatesUpdateById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchStatesUpdateById", h)
}

//export PatchUserConsultationSubscriptionsById
func PatchUserConsultationSubscriptionsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchUserConsultationSubscriptionsById", h)
}

//export PatchUsersChangePassword
func PatchUsersChangePassword(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchUsersChangePassword", h)
}

//export PatchUsersLawyersAcceptVerifiyingById
func PatchUsersLawyersAcceptVerifiyingById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchUsersLawyersAcceptVerifiyingById", h)
}

//export PatchUsersMe
func PatchUsersMe(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PatchUsersMe", h)
}

//export PostAdminReportsAction
func PostAdminReportsAction(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAdminReportsAction", h)
}

//export PostAssistants
func PostAssistants(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAssistants", h)
}

//export PostAuthLogin
func PostAuthLogin(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthLogin", h)
}

//export PostAuthLoginGoogle
func PostAuthLoginGoogle(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthLoginGoogle", h)
}

//export PostAuthLogout
func PostAuthLogout(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthLogout", h)
}

//export PostAuthRefreshToken
func PostAuthRefreshToken(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthRefreshToken", h)
}

//export PostAuthRegister
func PostAuthRegister(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthRegister", h)
}

//export PostAuthResendResetCode
func PostAuthResendResetCode(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthResendResetCode", h)
}

//export PostAuthResendVerificationCode
func PostAuthResendVerificationCode(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthResendVerificationCode", h)
}

//export PostAuthResetPassword
func PostAuthResetPassword(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthResetPassword", h)
}

//export PostAuthSetRole
func PostAuthSetRole(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthSetRole", h)
}

//export PostAuthVerifyEmail
func PostAuthVerifyEmail(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthVerifyEmail", h)
}

//export PostAuthVerifyResetCode
func PostAuthVerifyResetCode(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostAuthVerifyResetCode", h)
}

//export PostBanners
func PostBanners(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostBanners", h)
}

//export PostBotChat
func PostBotChat(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostBotChat", h)
}

//export PostCaseCategoriesStore
func PostCaseCategoriesStore(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCaseCategoriesStore", h)
}

//export PostCaseChambersStore
func PostCaseChambersStore(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCaseChambersStore", h)
}

//export PostCasePhasesStore
func PostCasePhasesStore(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasePhasesStore", h)
}

//export PostCasesByIdFileTitle
func PostCasesByIdFileTitle(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasesByIdFileTitle", h)
}

//export PostCasesByIdFiles
func PostCasesByIdFiles(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasesByIdFiles", h)
}

//export PostCasesByIdNotes
func PostCasesByIdNotes(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasesByIdNotes", h)
}

//export PostCasesByIdSaveCase
func PostCasesByIdSaveCase(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasesByIdSaveCase", h)
}

//export PostCasesByIdShare
func PostCasesByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasesByIdShare", h)
}

//export PostCasesStore
func PostCasesStore(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostCasesStore", h)
}

//export PostConsultationPackages
func PostConsultationPackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostConsultationPackages", h)
}

//export PostFeedsClientCases
func PostFeedsClientCases(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsClientCases", h)
}

//export PostFeedsClientCasesByIdApplicationsAcceptByApplicationId
func PostFeedsClientCasesByIdApplicationsAcceptByApplicationId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsClientCasesByIdApplicationsAcceptByApplicationId", h)
}

//export PostFeedsClientCasesByIdApplicationsRejectByApplicationId
func PostFeedsClientCasesByIdApplicationsRejectByApplicationId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsClientCasesByIdApplicationsRejectByApplicationId", h)
}

//export PostFeedsClientConsultations
func PostFeedsClientConsultations(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsClientConsultations", h)
}

//export PostFeedsClientConsultationsByIdAnswersByAnswerId
func PostFeedsClientConsultationsByIdAnswersByAnswerId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsClientConsultationsByIdAnswersByAnswerId", h)
}

//export PostFeedsLawyerCasesByIdApply
func PostFeedsLawyerCasesByIdApply(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsLawyerCasesByIdApply", h)
}

//export PostFeedsLawyerConsultationsByIdAnswers
func PostFeedsLawyerConsultationsByIdAnswers(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsLawyerConsultationsByIdAnswers", h)
}

//export PostFeedsLawyerConsultationsCommentsByAnswerId
func PostFeedsLawyerConsultationsCommentsByAnswerId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFeedsLawyerConsultationsCommentsByAnswerId", h)
}

//export PostFolders
func PostFolders(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFolders", h)
}

//export PostFoldersByIdFiles
func PostFoldersByIdFiles(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFoldersByIdFiles", h)
}

//export PostFoldersByIdShare
func PostFoldersByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFoldersByIdShare", h)
}

//export PostFoldersFilesByIdShare
func PostFoldersFilesByIdShare(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostFoldersFilesByIdShare", h)
}

//export PostForumPosts
func PostForumPosts(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostForumPosts", h)
}

//export PostForumPostsHide
func PostForumPostsHide(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostForumPostsHide", h)
}

//export PostForumPostsReport
func PostForumPostsReport(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostForumPostsReport", h)
}

//export PostForumReplies
func PostForumReplies(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostForumReplies", h)
}

//export PostForumRepliesReport
func PostForumRepliesReport(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostForumRepliesReport", h)
}

//export PostLawyerPackages
func PostLawyerPackages(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostLawyerPackages", h)
}

//export PostLawyerPackagesSubscriptionsCleanup
func PostLawyerPackagesSubscriptionsCleanup(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostLawyerPackagesSubscriptionsCleanup", h)
}

//export PostLawyerSubscriptionsSubscribe
func PostLawyerSubscriptionsSubscribe(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostLawyerSubscriptionsSubscribe", h)
}

//export PostPetitions
func PostPetitions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostPetitions", h)
}

//export PostPetitionsByIdMove
func PostPetitionsByIdMove(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostPetitionsByIdMove", h)
}

//export PostPetitionsImage
func PostPetitionsImage(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostPetitionsImage", h)
}

//export PostPetitionsUploadFile
func PostPetitionsUploadFile(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostPetitionsUploadFile", h)
}

//export PostRequests
func PostRequests(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostRequests", h)
}

//export PostRequestsByIdUpdateStatus
func PostRequestsByIdUpdateStatus(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostRequestsByIdUpdateStatus", h)
}

//export PostSpecializationsStore
func PostSpecializationsStore(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostSpecializationsStore", h)
}

//export PostStatesStore
func PostStatesStore(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostStatesStore", h)
}

//export PostUserConsultationSubscriptions
func PostUserConsultationSubscriptions(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUserConsultationSubscriptions", h)
}

//export PostUsersChatNotification
func PostUsersChatNotification(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersChatNotification", h)
}

//export PostUsersCreateImage
func PostUsersCreateImage(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersCreateImage", h)
}

//export PostUsersLawyerVerificationFiles
func PostUsersLawyerVerificationFiles(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersLawyerVerificationFiles", h)
}

//export PostUsersRemoveImage
func PostUsersRemoveImage(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersRemoveImage", h)
}

//export PostUsersRemoveImageCover
func PostUsersRemoveImageCover(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersRemoveImageCover", h)
}

//export PostUsersSavedAiImage
func PostUsersSavedAiImage(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersSavedAiImage", h)
}

//export PostUsersUploadImage
func PostUsersUploadImage(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersUploadImage", h)
}

//export PostUsersUploadImageCover
func PostUsersUploadImageCover(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PostUsersUploadImageCover", h)
}

//export PutCasesByIdRelations
func PutCasesByIdRelations(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PutCasesByIdRelations", h)
}

//export PutForumNotificationsByIdRead
func PutForumNotificationsByIdRead(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PutForumNotificationsByIdRead", h)
}

//export PutForumRepliesById
func PutForumRepliesById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PutForumRepliesById", h)
}

//export PutPetitionsById
func PutPetitionsById(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PutPetitionsById", h)
}

//export PutUsersLawyersAcceptByUserId
func PutUsersLawyersAcceptByUserId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PutUsersLawyersAcceptByUserId", h)
}

//export PutUsersLawyersRejectByUserId
func PutUsersLawyersRejectByUserId(e event.Event) uint32 {
	h, err := e.HTTP()
	if err != nil {
		return 1
	}
	return dispatchLawgenHandler("PutUsersLawyersRejectByUserId", h)
}
