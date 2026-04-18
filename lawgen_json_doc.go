package lib

import (
	"encoding/json"
	"sort"

	"github.com/taubyte/go-sdk/database"
)

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
