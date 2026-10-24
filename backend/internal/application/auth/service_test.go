package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

type fakeProvider struct {
	token       Token
	profile     Profile
	exchangeErr error
}

func (f fakeProvider) ExchangeCode(context.Context, string) (Token, error) {
	return f.token, f.exchangeErr
}

func (f fakeProvider) RefreshToken(context.Context, string) (Token, error) {
	return f.token, nil
}

func (f fakeProvider) Profile(context.Context, string) (Profile, error) {
	return f.profile, nil
}

type memStudents struct{ saved []student.Student }

func (m *memStudents) Save(_ context.Context, s student.Student) error {
	m.saved = append(m.saved, s)
	return nil
}

func TestLoginRegistersStudent(t *testing.T) {
	repo := &memStudents{}
	svc := NewService(fakeProvider{
		token:   Token{AccessToken: "access"},
		profile: Profile{ID: 42, Login: "jdoe", DisplayName: "John Doe"},
	}, repo)

	token, err := svc.Login(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if len(repo.saved) != 1 || repo.saved[0].Login != "jdoe" || repo.saved[0].ID != 42 {
		t.Errorf("saved = %+v", repo.saved)
	}
}

func TestLoginInvalidCodeSavesNothing(t *testing.T) {
	repo := &memStudents{}
	svc := NewService(fakeProvider{exchangeErr: ErrInvalidGrant}, repo)

	if _, err := svc.Login(context.Background(), "bad"); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("err = %v, want ErrInvalidGrant", err)
	}
	if len(repo.saved) != 0 {
		t.Errorf("saved = %+v, want none", repo.saved)
	}
}
