package lib

import (
	"strings"

	"github.com/taubyte/go-sdk/database"
	httpevent "github.com/taubyte/go-sdk/http/event"
)

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
