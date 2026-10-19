package main

import "testing"

func TestNotificationPayload(t *testing.T) {
	for _, tt := range []struct{ name, text string }{{"lifecycle", "Invite accepted"}, {"admin", "Member removed"}} {
		t.Run(tt.name, func(t *testing.T) {
			n := Notification{Text: tt.text}
			if n.Text != tt.text {
				t.Fatal("text changed")
			}
		})
	}
}
