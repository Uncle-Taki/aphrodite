package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testNow = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

func validUserDTO() UserDTO {
	phone := " 0912 "
	return UserDTO{
		ID:           uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"),
		Username:     "  sara ",
		Email:        " Sara@Example.COM ",
		PasswordHash: "hashed",
		PhoneNumber:  &phone,
		Role:         RoleEditor,
		Now:          testNow,
	}
}

func TestRole_Valid(t *testing.T) {
	for _, role := range []Role{RoleUser, RoleAdmin, RoleEditor} {
		if !role.Valid() {
			t.Errorf("expected role %q to be valid", role)
		}
	}
	for _, role := range []Role{"", "superuser", "guest"} {
		if role.Valid() {
			t.Errorf("expected role %q to be invalid", role)
		}
	}
}

func TestNewUser(t *testing.T) {
	user, err := NewUser(validUserDTO())
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "sara" {
		t.Errorf("username not trimmed: %q", user.Username)
	}
	if user.Email != "sara@example.com" {
		t.Errorf("email not normalized: %q", user.Email)
	}
	if user.PhoneNumber == nil || *user.PhoneNumber != "0912" {
		t.Errorf("phone not trimmed: %+v", user.PhoneNumber)
	}
	if user.CreatedAt != testNow || user.UpdatedAt != testNow {
		t.Errorf("timestamps not set from Now")
	}
}

func TestNewUserValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*UserDTO)
		want   error
	}{
		{"empty username", func(d *UserDTO) { d.Username = "   " }, ErrInvalidUsername},
		{"email without at", func(d *UserDTO) { d.Email = "sara.example.com" }, ErrInvalidEmail},
		{"email without domain dot", func(d *UserDTO) { d.Email = "sara@localhost" }, ErrInvalidEmail},
		{"email with at first", func(d *UserDTO) { d.Email = "@example.com" }, ErrInvalidEmail},
		{"email with at last", func(d *UserDTO) { d.Email = "sara@" }, ErrInvalidEmail},
		{"empty password hash", func(d *UserDTO) { d.PasswordHash = "" }, ErrInvalidPasswordHash},
		{"invalid role", func(d *UserDTO) { d.Role = "root" }, ErrInvalidRole},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dto := validUserDTO()
			tc.mutate(&dto)
			_, err := NewUser(dto)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestNewUserTrimsEmptyPhoneToNil(t *testing.T) {
	empty := "   "
	dto := validUserDTO()
	dto.PhoneNumber = &empty
	user, err := NewUser(dto)
	if err != nil {
		t.Fatal(err)
	}
	if user.PhoneNumber != nil {
		t.Fatalf("expected nil phone, got %q", *user.PhoneNumber)
	}
}

func TestUser_Update(t *testing.T) {
	user, err := NewUser(validUserDTO())
	if err != nil {
		t.Fatal(err)
	}
	phone := "+98912"
	later := testNow.Add(time.Hour)
	if err := user.Update(UpdateDTO{
		Username:    " sara2 ",
		Email:       " Sara2@Example.com ",
		PhoneNumber: &phone,
		Role:        RoleAdmin,
		Disabled:    true,
		Now:         later,
	}); err != nil {
		t.Fatal(err)
	}
	if user.Username != "sara2" || user.Email != "sara2@example.com" {
		t.Errorf("fields not normalized: %+v", user)
	}
	if user.Role != RoleAdmin || !user.Disabled {
		t.Errorf("role/disabled not applied: %+v", user)
	}
	if user.UpdatedAt != later {
		t.Errorf("UpdatedAt not bumped")
	}
	if user.PhoneNumber == nil || *user.PhoneNumber != "+98912" {
		t.Errorf("phone not applied: %+v", user.PhoneNumber)
	}
}

func TestUser_UpdateValidation(t *testing.T) {
	user, _ := NewUser(validUserDTO())
	cases := []struct {
		name string
		dto  UpdateDTO
		want error
	}{
		{"empty username", UpdateDTO{Username: " ", Email: "a@b.co", Role: RoleUser}, ErrInvalidUsername},
		{"bad email", UpdateDTO{Username: "sara", Email: "nope", Role: RoleUser}, ErrInvalidEmail},
		{"bad role", UpdateDTO{Username: "sara", Email: "a@b.co", Role: "x"}, ErrInvalidRole},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := user.Update(tc.dto); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestUser_UpdateBlanksPhone(t *testing.T) {
	user, _ := NewUser(validUserDTO())
	blank := "  "
	if err := user.Update(UpdateDTO{Username: "sara", Email: "a@b.co", PhoneNumber: &blank, Role: RoleUser}); err != nil {
		t.Fatal(err)
	}
	if user.PhoneNumber != nil {
		t.Fatalf("expected phone cleared, got %q", *user.PhoneNumber)
	}
}

func TestUser_ChangePassword(t *testing.T) {
	user, _ := NewUser(validUserDTO())
	if err := user.ChangePassword("", testNow); !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("expected ErrInvalidPasswordHash, got %v", err)
	}
	later := testNow.Add(time.Minute)
	if err := user.ChangePassword("new-hash", later); err != nil {
		t.Fatal(err)
	}
	if user.PasswordHash != "new-hash" || user.UpdatedAt != later {
		t.Errorf("password/timestamp not applied: %+v", user)
	}
}

func TestLooksLikeEmail(t *testing.T) {
	cases := map[string]bool{
		"a@b.co":  true,
		"a@bc.co": true,
		"a@b@c.d": true, // naive check accepts multiple ats
		"b.co":    false,
		"@b.co":   false,
		"a@":      false,
		"a@bco":   false,
		"":        false,
		"a @b.co": true,
	}
	for email, want := range cases {
		if got := looksLikeEmail(email); got != want {
			t.Errorf("looksLikeEmail(%q) = %v, want %v", email, got, want)
		}
	}
}
