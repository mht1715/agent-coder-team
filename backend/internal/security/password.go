package security

import "github.com/alexedwards/argon2id"

type PasswordHasher struct {
	params *argon2id.Params
}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{params: argon2id.DefaultParams}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	return argon2id.CreateHash(password, h.params)
}

func (h *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, encodedHash)
}
