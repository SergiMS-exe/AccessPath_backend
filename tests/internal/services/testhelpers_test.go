package services_test

import (
	"golang.org/x/crypto/bcrypt"
)

// bcryptHash devuelve un hash bcrypt del password para usar en tests que
// ejercitan la verificacion real de credenciales (Login).
func bcryptHash(t testingT, password string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

// testingT evita importar testing en cada sitio donde se usa bcryptHash.
type testingT interface {
	Fatalf(format string, args ...any)
}
