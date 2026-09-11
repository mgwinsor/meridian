package cash

import (
	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/money"
)

type Balance struct {
	AccountID account.ID
	Amount    money.Amount
}
