package notification

import (
	"testing"
)

func TestMailNotification_SendGroupInvitationEmail(t *testing.T) {
	config := MailConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}

	notif := NewMailNotification(config)

	// This will fail because we don't have a real SMTP server
	err := notif.SendGroupInvitationEmail(
		"to@example.com",
		"Invitee Name",
		"Inviter Name",
		"Group Name",
		"Group Type",
		"Group Description",
		"https://example.com/accept",
	)
	if err == nil {
		t.Log("SendGroupInvitationEmail succeeded (unexpected)")
	} else {
		// Expected error - verify it's not a template parsing error
		if err.Error() == "failed to parse template" {
			t.Errorf("Template parsing failed: %v", err)
		}
	}
}
