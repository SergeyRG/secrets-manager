package cli

type SecretAuthData struct {
	Login    string
	Password string
}

type SecretBankData struct {
	CardNum string
	CVC     string
}

type SecretFreeText struct {
	Data string
}
