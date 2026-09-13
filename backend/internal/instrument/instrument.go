package instrument

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/mgwinsor/meridian/backend/internal/currency"
)

var (
	ErrInvalidID     = errors.New("invalid instrument ID")
	ErrInvalidKind   = errors.New("unsupported instrument kind")
	ErrInvalidSymbol = errors.New("instrument symbol cannot be empty")
	ErrInvalidName   = errors.New("instrument name cannot be empty")
	ErrNotFound      = errors.New("instrument not found")
)

type ID struct{ value uuid.UUID }

func NewID() ID { return ID{value: uuid.New()} }

func ParseID(value string) (ID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id.String() != value {
		return ID{}, ErrInvalidID
	}
	return ID{value: id}, nil
}

func (id ID) String() string { return id.value.String() }

type Kind string

const (
	KindStock      Kind = "stock"
	KindETF        Kind = "etf"
	KindBond       Kind = "bond"
	KindMutualFund Kind = "mutual_fund"
	KindCrypto     Kind = "crypto"
)

// Instrument describes an investment vehicle independently of ownership or valuation.
type Instrument struct {
	ID            ID
	Kind          Kind
	Symbol        string
	Name          string
	QuoteCurrency currency.Code
}

func New(id ID, kind Kind, symbol, name string, quoteCurrency currency.Code) (Instrument, error) {
	switch kind {
	case KindStock, KindETF, KindBond, KindMutualFund, KindCrypto:
	default:
		return Instrument{}, ErrInvalidKind
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return Instrument{}, ErrInvalidSymbol
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Instrument{}, ErrInvalidName
	}
	if _, err := currency.Parse(quoteCurrency.String()); err != nil {
		return Instrument{}, err
	}
	return Instrument{ID: id, Kind: kind, Symbol: symbol, Name: name, QuoteCurrency: quoteCurrency}, nil
}
