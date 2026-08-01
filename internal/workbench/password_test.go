package workbench

import (
	"testing"
)

func TestVerifyPassword(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !verifyPassword(hash, "s3cret-pass") {
		t.Fatal("correct password must verify")
	}
	if verifyPassword(hash, "s3cret-pas") {
		t.Fatal("wrong password must fail")
	}
	if verifyPassword(hash, "") {
		t.Fatal("empty password must fail")
	}
}

func TestVerifyPassword_RejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"plaintext",
		"pbkdf2$0$c2FsdA==$aGVsbG8=",      // iter=0
		"pbkdf2$-1$c2FsdA==$aGVsbG8=",     // iter<0
		"pbkdf2$abc$c2FsdA==$aGVsbG8=",    // iter 非数字
		"pbkdf2$999999999999$c2FsdA==$aGVsbG8=", // iter 超上限
		"pbkdf2$1000$$aGVsbG8=",           // 盐为空
		"pbkdf2$1000$c2FsdA==$",           // hash 为空
		"pbkdf2$1000$c2FsdA==$!!!",        // hash 非 base64
		"md5$1000$c2FsdA==$aGVsbG8=",      // 算法前缀错误
		"pbkdf2$1000$c2FsdA==$aGVsbG8=$extra", // 段数错误
	}
	for _, c := range cases {
		if verifyPassword(c, "any-password") {
			t.Errorf("malformed hash %q must fail verification", c)
		}
	}
}

func TestHashPassword_SaltRandomness(t *testing.T) {
	h1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Fatal("same password must produce different hashes (random salt)")
	}
}
