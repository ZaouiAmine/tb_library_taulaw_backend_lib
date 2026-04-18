package lib

import (
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/taubyte/go-sdk/database"
	httpevent "github.com/taubyte/go-sdk/http/event"
)

const lawgenKVMatcher = "taulaw_kv"

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
	return database.New(lawgenKVMatcher)
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
