package backend

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	s := "ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvxyz-"
	if ss := sanitizeName(s); ss != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = "this-IS-a-ReAlLy-COMPLEX-name-that-SHOULD-be-LEFT-unTOUCHED"
	if ss := sanitizeName(s); ss != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName(" this has a_leading*illegal(char")
	if "-this-has-a-leading-illegal-char" != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName("this has a_trailing*illegal(char=")
	if "this-has-a-trailing-illegal-char-" != s {
		t.Errorf("Resulting string was: %s", s)
	}

	s = sanitizeName("Some Chinese: 波前的岩石")
	if "Some-Chinese-------" != s {
		t.Errorf("Resulting string was: %s", s)
	}
}
