package dto

type SearchPatientsRequest struct {
	NationalID  *string `json:"national_id" binding:"omitempty,max=20"`
	PassportID  *string `json:"passport_id" binding:"omitempty,max=32"`
	FirstName   *string `json:"first_name" binding:"omitempty,max=255"`
	MiddleName  *string `json:"middle_name" binding:"omitempty,max=255"`
	LastName    *string `json:"last_name" binding:"omitempty,max=255"`
	DateOfBirth *string `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	PhoneNumber *string `json:"phone_number" binding:"omitempty,max=32"`
	Email       *string `json:"email" binding:"omitempty,email,max=320"`
	Page        int     `json:"page" binding:"omitempty,min=1"`
	PageSize    int     `json:"page_size" binding:"omitempty,min=1,max=100"`
}

type SearchPatientsResponse struct {
	Data       []PatientData `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

type PatientData struct {
	FirstNameTH  *string `json:"first_name_th"`
	MiddleNameTH *string `json:"middle_name_th"`
	LastNameTH   *string `json:"last_name_th"`
	FirstNameEN  *string `json:"first_name_en"`
	MiddleNameEN *string `json:"middle_name_en"`
	LastNameEN   *string `json:"last_name_en"`
	DateOfBirth  *string `json:"date_of_birth"`
	PatientHN    string  `json:"patient_hn"`
	NationalID   *string `json:"national_id"`
	PassportID   *string `json:"passport_id"`
	PhoneNumber  *string `json:"phone_number"`
	Email        *string `json:"email"`
	Gender       *string `json:"gender"`
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}
