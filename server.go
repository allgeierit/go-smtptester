// Package smtptester implements a simple SMTP server for testing. All
// received mails are saved in a sync.Map with a key:
//
//      From+Recipient1+Recipient2
//
// Mails to the same sender and recipients will overwrite a previous
// received mail, when the recipients slice has the same order as
// in the mail received before.
package smtptester

import (
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
)

// Standard returns a standard SMTP server listening on :2525
func Standard() *smtp.Server {
	srv := smtp.NewServer(NewBackend())

	srv.Addr = ":2525"
	srv.Domain = "127.0.0.1"
	srv.ReadTimeout = 10 * time.Second
	srv.WriteTimeout = 10 * time.Second
	srv.MaxMessageBytes = 1024 * 1024
	srv.MaxRecipients = 100
	srv.AllowInsecureAuth = true

	return srv
}

///////////////////////////////////////////////////////////////////////////
// Backend
///////////////////////////////////////////////////////////////////////////

// Backend is the backend for out test server. It contains a sync.Map
// with all mails received.
type Backend struct {
	Mails sync.Map
}

// NewBackend returns a new Backend with an empty (not nil) Mails map.
func NewBackend() *Backend {
	return &Backend{Mails: sync.Map{}}
}

// GetBackend returns the concrete type *Backend from SMTP server.
func GetBackend(s *smtp.Server) *Backend {
	if s.Backend == nil {
		return nil
	}

	b, _ := s.Backend.(*Backend)
	return b
}

// Add adds mail to backends map.
func (b *Backend) Add(m *Mail) {
	b.Mails.Store(m.LookupKey(), m)
}

// Load loads mail from 'from' to recipients 'recipients'. The ok
// result indicates whether value was found in the map.
func (b *Backend) Load(from string, recipients []string) (*Mail, bool) {
	i, ok := b.Mails.Load(LookupKey(from, recipients))
	if !ok {
		return nil, ok
	}

	return i.(*Mail), ok
}

// Login implements Login interface.
func (b *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return newSession(b), nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (b *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return newSession(b), nil
}

///////////////////////////////////////////////////////////////////////////
// Session
///////////////////////////////////////////////////////////////////////////

// A Session is returned after successful login.
type Session struct {
	backend *Backend
	mail    *Mail
}

func newSession(b *Backend) *Session {
	return &Session{
		backend: b,
		mail:    &Mail{},
	}
}

// Mail implements the Mail interface.
func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	s.mail.From = from

	return nil
}

// Rcpt implements the Rcpt interface.
func (s *Session) Rcpt(to string) error {
	s.mail.Recipients = append(s.mail.Recipients, to)

	return nil
}

// Data implements the Data interface.
func (s *Session) Data(r io.Reader) error {
	var err error

	if s.mail.Data, err = ioutil.ReadAll(r); err != nil {
		return err
	}

	s.backend.Add(s.mail)

	return nil
}

// Reset implements Reset interface.
func (s *Session) Reset() {
	s.mail = &Mail{}
}

// Logout implements Logout interface.
func (s *Session) Logout() error {
	return nil
}
