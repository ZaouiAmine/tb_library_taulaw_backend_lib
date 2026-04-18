package lib

import (
	"encoding/json"
	"io"
	"strings"

	httpevent "github.com/taubyte/go-sdk/http/event"
)

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
