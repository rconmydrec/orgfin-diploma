package models

import "testing"

func TestDisplayName(t *testing.T) {
	first := "John"
	last := "Doe"

	tests := []struct {
		name      string
		firstName *string
		lastName  *string
		want      string
	}{
		{"both names", &first, &last, "John Doe"},
		{"first only", &first, nil, "John"},
		{"last only", nil, &last, "Doe"},
		{"both nil", nil, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{FirstName: tt.firstName, LastName: tt.lastName}
			if got := u.DisplayName(); got != tt.want {
				t.Errorf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}
