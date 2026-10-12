package assetclass

import "errors"

type Class string

const (
	Cash         Class = "cash"
	Equities     Class = "equities"
	Bonds        Class = "bonds"
	Property     Class = "property"
	Crypto       Class = "crypto"
	Mixed        Class = "mixed"
	Unclassified Class = "unclassified"
)

var ErrInvalid = errors.New("unsupported investment asset class")

func ParseInvestment(value string) (Class, error) {
	switch Class(value) {
	case Equities, Bonds, Crypto, Mixed, Unclassified:
		return Class(value), nil
	}
	return "", ErrInvalid
}
