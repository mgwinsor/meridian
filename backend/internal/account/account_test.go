package account_test

import (
	"testing"

	"github.com/mgwinsor/meridian/backend/internal/account"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name            string
		accountName     string
		wantAccountName string
		wantErr         bool
	}{
		{
			name:            "valid account",
			accountName:     "account 1",
			wantAccountName: "account 1",
			wantErr:         false,
		},
		{
			name:            "strip surrounding whitespace",
			accountName:     "   account 1   ",
			wantAccountName: "account 1",
			wantErr:         false,
		},
		{
			name:        "empty name",
			accountName: "",
			wantErr:     true,
		},
		{
			name:        "whitespace only name",
			accountName: "   ",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := account.NewID()

			got, err := account.New(id, tt.accountName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("New() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}

			if got.ID != id {
				t.Errorf("New() ID = %v, want %v", got.ID, id)
			}

			if got.Name != tt.wantAccountName {
				t.Errorf("New() Name = %q, want %q", got.Name, tt.wantAccountName)
			}
		})
	}
}
