package usecases

type UserAuthDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
