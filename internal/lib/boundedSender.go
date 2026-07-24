package lib

import (
	"text/template"

	"github.com/a-novel-kit/golib/smtp"
)

// BoundedSender is a [smtp.Sender] that caps how many deliveries run at once, delaying further
// sends until a slot frees.
//
// Every delivery runs on its own detached goroutine and holds one connection to the SMTP server
// for up to its full timeout. The cap keeps a burst of requests — or a server that stops
// answering — from piling up connections.
type BoundedSender struct {
	sender smtp.Sender
	slots  chan struct{}
}

// NewBoundedSender wraps sender so at most limit deliveries run concurrently. A limit below one is
// raised to one.
func NewBoundedSender(sender smtp.Sender, limit int) *BoundedSender {
	return &BoundedSender{sender: sender, slots: make(chan struct{}, max(limit, 1))}
}

// SendMail waits for a delivery slot, then delegates to the wrapped sender.
func (bounded *BoundedSender) SendMail(to smtp.MailUsers, t *template.Template, tName string, data any) error {
	bounded.slots <- struct{}{}
	defer func() { <-bounded.slots }()

	return bounded.sender.SendMail(to, t, tName, data)
}

// Ping delegates to the wrapped sender without taking a delivery slot, so readiness answers even
// while every slot is busy.
func (bounded *BoundedSender) Ping() error {
	return bounded.sender.Ping()
}
