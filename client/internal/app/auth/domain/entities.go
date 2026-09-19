package domain

type TokenStorage interface {
	GetToken() string
	SetToken() string
}

// type Creds struct {
// 	Username string
// 	Passwd   string
// }

// var alphaNumericUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// var (
// 	ErrInvalidNameFormat error = errors.New("неправильный формат имени")
// )

// func (c Creds) ValidateName() error {
// 	if !alphaNumericUnderscoreRegex.MatchString(c.Username) {
// 		return ErrInvalidNameFormat
// 	}
// 	return nil
// }
