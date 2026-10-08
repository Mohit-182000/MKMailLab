package bridge

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"localmail/internal/domain"
	"localmail/internal/events"
	"localmail/internal/mailbox"
	"localmail/internal/smtpd"
)

// Registering event payload types lets the binding generator emit typed
// event definitions for the frontend.
func init() {
	application.RegisterEvent[smtpd.Status](events.SMTPStatus)
	application.RegisterEvent[domain.MessageSummary](events.MailReceived)
	application.RegisterEvent[mailbox.Change](events.MailChanged)
}
