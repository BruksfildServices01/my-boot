package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const cookieName = "myboot_admin"

type Store struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]time.Time)}
}

func (s *Store) Create(w http.ResponseWriter) {
	id := randomID()
	s.mu.Lock()
	s.sessions[id] = time.Now().Add(24 * time.Hour)
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Store) IsValid(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	s.mu.RLock()
	exp, ok := s.sessions[c.Value]
	s.mu.RUnlock()
	return ok && time.Now().Before(exp)
}

func (s *Store) Destroy(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
