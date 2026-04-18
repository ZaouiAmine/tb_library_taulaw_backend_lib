package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/taubyte/go-sdk/database"
	"github.com/taubyte/go-sdk/event"
	httpevent "github.com/taubyte/go-sdk/http/event"
)

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
