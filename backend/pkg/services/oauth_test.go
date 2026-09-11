package services

import "testing"

func TestOIDCRoleForEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		adminEmails []string
		want        string
	}{
		{
			name:        "configured administrator",
			email:       "admin@example.com",
			adminEmails: []string{"admin@example.com"},
			want:        "admin",
		},
		{
			name:        "email comparison ignores case and surrounding spaces",
			email:       " Admin@Example.com ",
			adminEmails: []string{"other@example.com", " admin@example.com "},
			want:        "admin",
		},
		{
			name:        "regular user",
			email:       "user@example.com",
			adminEmails: []string{"admin@example.com"},
			want:        "none",
		},
		{
			name:        "empty administrator list",
			email:       "admin@example.com",
			adminEmails: nil,
			want:        "none",
		},
		{
			name:        "empty email is never an administrator",
			email:       " ",
			adminEmails: []string{" "},
			want:        "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := oidcRoleForEmail(tt.email, tt.adminEmails); got != tt.want {
				t.Fatalf("oidcRoleForEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}
