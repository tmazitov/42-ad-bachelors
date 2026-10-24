package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/session"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

type fakeProvider struct {
	profile     student.Student
	exchangeErr error
}

func (f fakeProvider) AuthorizeURL(state string) string { return "https://42/authorize?state=" + state }

func (f fakeProvider) ExchangeCode(context.Context, string) (Token, error) {
	return Token{AccessToken: "user-token"}, f.exchangeErr
}

func (f fakeProvider) Profile(context.Context, string) (student.Student, error) {
	return f.profile, nil
}

// memStudents implements only what auth uses.
type memStudents struct {
	student.Repository
	saved []student.Student
}

func (m *memStudents) Save(_ context.Context, s student.Student) error {
	m.saved = append(m.saved, s)
	return nil
}

type memSessions struct{ byHash map[string]session.Session }

func newMemSessions() *memSessions { return &memSessions{byHash: map[string]session.Session{}} }

func (m *memSessions) Create(_ context.Context, s session.Session) error {
	m.byHash[s.TokenHash] = s
	return nil
}

func (m *memSessions) Find(_ context.Context, hash string) (session.Session, error) {
	s, ok := m.byHash[hash]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	return s, nil
}

func (m *memSessions) Delete(_ context.Context, hash string) error {
	delete(m.byHash, hash)
	return nil
}

func (m *memSessions) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }

type fakeSyncer struct {
	synced []student.ID
	err    error
}

func (f *fakeSyncer) SyncStudent(_ context.Context, _ string, id student.ID) error {
	f.synced = append(f.synced, id)
	return f.err
}

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func newService(p fakeProvider, students *memStudents, sessions *memSessions, syncer *fakeSyncer) *Service {
	return NewService(p, students, sessions, syncer, 24*time.Hour, discard)
}

func TestLoginRegistersStudentAndOpensSession(t *testing.T) {
	students, sessions := &memStudents{}, newMemSessions()
	// A failing sync must not fail the login.
	syncer := &fakeSyncer{err: errors.New("42 down")}
	svc := newService(fakeProvider{profile: student.Student{ID: 42, Login: "jdoe", DisplayName: "John Doe"}},
		students, sessions, syncer)

	token, expires, err := svc.Login(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || time.Until(expires) < 23*time.Hour {
		t.Errorf("token = %q, expires = %v", token, expires)
	}
	if len(students.saved) != 1 || students.saved[0].ID != 42 || !students.saved[0].Registered() {
		t.Errorf("saved = %+v, want registered student 42", students.saved)
	}
	if len(syncer.synced) != 1 || syncer.synced[0] != 42 {
		t.Errorf("synced = %v, want [42]", syncer.synced)
	}

	// The session token authenticates; only its hash is stored.
	if _, ok := sessions.byHash[token]; ok {
		t.Error("raw token stored")
	}
	id, err := svc.Authenticate(context.Background(), token)
	if err != nil || id != 42 {
		t.Errorf("Authenticate = %d, %v; want 42", id, err)
	}

	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after logout err = %v, want ErrUnauthenticated", err)
	}
}

func TestLoginInvalidCode(t *testing.T) {
	students, sessions := &memStudents{}, newMemSessions()
	svc := newService(fakeProvider{exchangeErr: ErrInvalidGrant}, students, sessions, &fakeSyncer{})

	if _, _, err := svc.Login(context.Background(), "bad"); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("err = %v, want ErrInvalidGrant", err)
	}
	if len(students.saved) != 0 || len(sessions.byHash) != 0 {
		t.Errorf("saved %d students and %d sessions, want none", len(students.saved), len(sessions.byHash))
	}
}

func TestAuthenticateRejectsUnknownAndExpired(t *testing.T) {
	sessions := newMemSessions()
	svc := newService(fakeProvider{}, &memStudents{}, sessions, &fakeSyncer{})

	token, sess, _ := session.New(1, time.Hour, time.Now().Add(-2*time.Hour)) // expired an hour ago
	sessions.Create(context.Background(), sess)

	for _, tok := range []string{"", "unknown", token} {
		if _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("Authenticate(%q) err = %v, want ErrUnauthenticated", tok, err)
		}
	}
	if len(sessions.byHash) != 0 {
		t.Error("expired session not deleted")
	}
}
