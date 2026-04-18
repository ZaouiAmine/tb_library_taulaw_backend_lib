package lib

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/taubyte/go-sdk/database"
	httpevent "github.com/taubyte/go-sdk/http/event"
)

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
