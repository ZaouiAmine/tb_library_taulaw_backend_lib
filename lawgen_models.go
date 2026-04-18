package lib

// kvUser mirrors essential fields from the Nest user entity for API responses.
type kvUser struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	Address              string `json:"address"`
	Role                 int    `json:"role"`
	Status               int    `json:"status"`
	ImgURL               string `json:"imgUrl,omitempty"`
	ImageCover           string `json:"imageCover,omitempty"`
	Description          string `json:"description,omitempty"`
	IsEmailVerified      bool   `json:"isEmailVerified"`
	CreatedByLawyerID    string `json:"createdByLawyerId,omitempty"`
	VerificationStatus   int    `json:"verificationStatus"`
	PasswordHash         string `json:"-"`
	RefreshToken         string `json:"-"`
	FcmToken             string `json:"-"`
}

func (u kvUser) publicDTO() map[string]any {
	return map[string]any{
		"id":                   u.ID,
		"name":                 u.Name,
		"email":                u.Email,
		"phone":                u.Phone,
		"address":              u.Address,
		"role":                 u.Role,
		"status":               u.Status,
		"imgUrl":               u.ImgURL,
		"imageCover":           u.ImageCover,
		"description":          u.Description,
		"isEmailVerified":      u.IsEmailVerified,
		"createdByLawyerId":    u.CreatedByLawyerID,
		"verificationStatus":   u.VerificationStatus,
	}
}

type registerBody struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	Name              string `json:"name"`
	Phone             string `json:"phone"`
	StateID           string `json:"stateId"`
	SpecializationID  string `json:"specializationId"`
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
