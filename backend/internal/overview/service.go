// Package overview values a consistent input snapshot without owning source storage.
package overview

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/account"
	"github.com/mgwinsor/meridian/backend/internal/assetclass"
	"github.com/mgwinsor/meridian/backend/internal/cash"
	"github.com/mgwinsor/meridian/backend/internal/currency"
	"github.com/mgwinsor/meridian/backend/internal/fx"
	"github.com/mgwinsor/meridian/backend/internal/instrument"
	"github.com/mgwinsor/meridian/backend/internal/position"
	"github.com/mgwinsor/meridian/backend/internal/price"
	"github.com/mgwinsor/meridian/backend/internal/property"
	"github.com/shopspring/decimal"
)

// Inputs contains current source records and the latest eligible observations,
// already selected by their owning repositories at the reader cutoff.
type Inputs struct {
	Accounts    []account.Account
	Cash        []cash.Balance
	Instruments []instrument.Instrument
	Positions   []position.Position
	Properties  []property.Property
	Prices      []price.Observation
	Rates       []fx.Observation
}

// InputReader returns all inputs from one consistent storage snapshot.
type InputReader interface {
	Read(context.Context, time.Time) (Inputs, error)
}

type Service struct{ reader InputReader }

func NewService(reader InputReader) Service { return Service{reader} }

func (s Service) Generate(ctx context.Context, code currency.Code) (Report, error) {
	at := time.Now().UTC().Truncate(time.Microsecond)
	inputs, err := s.reader.Read(ctx, at)
	if err != nil {
		return Report{}, err
	}
	return Calculate(inputs, code, at)
}

type SelectedPrice struct {
	Amount     string    `json:"amount"`
	Currency   string    `json:"currency"`
	ObservedAt time.Time `json:"observedAt"`
}

type UsedRate struct {
	Currency   string    `json:"currency"`
	Rate       string    `json:"rate"`
	ObservedAt time.Time `json:"observedAt"`
}

type Asset struct {
	Source         string           `json:"source"`
	ID             string           `json:"id"`
	AccountID      string           `json:"accountId,omitempty"`
	InstrumentID   string           `json:"instrumentId,omitempty"`
	Name           string           `json:"name"`
	AssetClass     assetclass.Class `json:"assetClass"`
	Currency       string           `json:"currency"`
	Quantity       *string          `json:"quantity,omitempty"`
	NativeValue    *string          `json:"nativeValue,omitempty"`
	ConvertedValue *string          `json:"convertedValue,omitempty"`
	Price          *SelectedPrice   `json:"price,omitempty"`
	MissingInputs  []string         `json:"missingInputs"`
}

type Breakdown struct {
	Key            string  `json:"key"`
	Amount         string  `json:"amount"`
	Percentage     *string `json:"percentage,omitempty"`
	ExcludedAssets int     `json:"excludedAssets"`
}

type Report struct {
	Currency       string      `json:"currency"`
	GeneratedAt    time.Time   `json:"generatedAt"`
	Complete       bool        `json:"complete"`
	ValuedTotal    string      `json:"valuedTotal"`
	Assets         []Asset     `json:"assets"`
	AssetClasses   []Breakdown `json:"assetClasses"`
	Currencies     []Breakdown `json:"currencies"`
	FXObservations []UsedRate  `json:"fxObservations"`
}

func pointer(s string) *string { return &s }

func Calculate(inputs Inputs, target currency.Code, at time.Time) (Report, error) {
	digits, err := target.MinorUnitDigits()
	if err != nil {
		return Report{}, err
	}
	precision := int32(digits)
	report := Report{
		Currency:       target.String(),
		GeneratedAt:    at,
		Complete:       true,
		Assets:         []Asset{},
		AssetClasses:   []Breakdown{},
		Currencies:     []Breakdown{},
		FXObservations: []UsedRate{},
	}
	accounts := map[account.ID]string{}
	for _, a := range inputs.Accounts {
		accounts[a.ID] = a.Name
	}
	instruments := map[instrument.ID]instrument.Instrument{}
	for _, i := range inputs.Instruments {
		instruments[i.ID] = i
	}
	prices := map[instrument.ID]price.Observation{}
	for _, p := range inputs.Prices {
		prices[p.InstrumentID] = p
	}
	rates := map[string]fx.Observation{}
	for _, r := range inputs.Rates {
		rates[r.Currency.String()] = r
	}
	for _, b := range inputs.Cash {
		report.Assets = append(report.Assets, Asset{
			Source:        "cash",
			ID:            b.AccountID.String() + ":" + b.Amount.Currency().String(),
			AccountID:     b.AccountID.String(),
			Name:          accounts[b.AccountID],
			AssetClass:    assetclass.Cash,
			Currency:      b.Amount.Currency().String(),
			NativeValue:   pointer(b.Amount.String()),
			MissingInputs: []string{},
		})
	}
	for _, p := range inputs.Properties {
		report.Assets = append(report.Assets, Asset{
			Source:        "property",
			ID:            p.ID.String(),
			Name:          p.Name,
			AssetClass:    assetclass.Property,
			Currency:      p.Value.Currency().String(),
			NativeValue:   pointer(p.Value.String()),
			MissingInputs: []string{},
		})
	}
	for _, p := range inputs.Positions {
		i, ok := instruments[p.InstrumentID]
		if !ok {
			return Report{}, fmt.Errorf("position references missing instrument %s", p.InstrumentID)
		}
		a := Asset{
			Source:        "position",
			ID:            p.AccountID.String() + ":" + p.InstrumentID.String(),
			AccountID:     p.AccountID.String(),
			InstrumentID:  p.InstrumentID.String(),
			Name:          accounts[p.AccountID] + " · " + i.Name,
			AssetClass:    i.AssetClass,
			Currency:      i.QuoteCurrency.String(),
			Quantity:      pointer(p.Quantity.String()),
			MissingInputs: []string{},
		}
		quantity, _ := decimal.NewFromString(p.Quantity.String())
		if observation, ok := prices[p.InstrumentID]; ok {
			if observation.Amount.Currency() != i.QuoteCurrency {
				return Report{}, fmt.Errorf("price currency does not match instrument")
			}
			a.Price = &SelectedPrice{
				Amount:     observation.Amount.String(),
				Currency:   observation.Amount.Currency().String(),
				ObservedAt: observation.ObservedAt,
			}
			amount, _ := decimal.NewFromString(observation.Amount.String())
			a.NativeValue = pointer(quantity.Mul(amount).String())
		} else if quantity.IsZero() {
			a.NativeValue = pointer("0")
		} else {
			a.MissingInputs = append(a.MissingInputs, "price")
		}
		report.Assets = append(report.Assets, a)
	}
	sort.Slice(report.Assets, func(i, j int) bool {
		a, b := report.Assets[i], report.Assets[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.ID < b.ID
	})
	used := map[string]bool{}
	total := decimal.Zero
	for index := range report.Assets {
		a := &report.Assets[index]
		if a.NativeValue == nil {
			report.Complete = false
			continue
		}
		value, _ := decimal.NewFromString(*a.NativeValue)
		if !value.IsZero() && a.Currency != target.String() {
			sourceRate, targetRate := decimal.NewFromInt(1), decimal.NewFromInt(1)
			for _, code := range []string{a.Currency, target.String()} {
				if code == "USD" {
					continue
				}
				rate, ok := rates[code]
				if !ok {
					a.MissingInputs = append(a.MissingInputs, "fx:"+code)
					continue
				}
				parsed, _ := decimal.NewFromString(rate.Rate.String())
				if code == a.Currency {
					sourceRate = parsed
				} else {
					targetRate = parsed
				}
			}
			if len(a.MissingInputs) > 0 {
				report.Complete = false
				continue
			}
			// Round the exact quotient directly to target precision,
			// without intermediate division rounding.
			value = value.Mul(targetRate).DivRound(sourceRate, precision)
			for _, code := range []string{a.Currency, target.String()} {
				if code != "USD" {
					used[code] = true
				}
			}
		} else {
			value = value.Round(precision)
		}
		a.ConvertedValue = pointer(value.StringFixed(precision))
		total = total.Add(value)
	}
	report.ValuedTotal = total.StringFixed(precision)
	report.AssetClasses = breakdown(report.Assets, total, precision, func(a Asset) string {
		return string(a.AssetClass)
	})
	report.Currencies = breakdown(report.Assets, total, precision, func(a Asset) string {
		return a.Currency
	})
	for code := range used {
		r := rates[code]
		report.FXObservations = append(report.FXObservations, UsedRate{code, r.Rate.String(), r.ObservedAt})
	}
	sort.Slice(report.FXObservations, func(i, j int) bool {
		return report.FXObservations[i].Currency < report.FXObservations[j].Currency
	})
	return report, nil
}

func breakdown(assets []Asset, total decimal.Decimal, precision int32, key func(Asset) string) []Breakdown {
	amounts := map[string]decimal.Decimal{}
	excluded := map[string]int{}
	for _, a := range assets {
		k := key(a)
		if _, ok := amounts[k]; !ok {
			amounts[k] = decimal.Zero
		}
		if a.ConvertedValue == nil {
			excluded[k]++
		} else {
			value, _ := decimal.NewFromString(*a.ConvertedValue)
			amounts[k] = amounts[k].Add(value)
		}
	}
	rows := []Breakdown{}
	for k, value := range amounts {
		row := Breakdown{Key: k, Amount: value.StringFixed(precision), ExcludedAssets: excluded[k]}
		if !total.IsZero() {
			row.Percentage = pointer(value.Mul(decimal.NewFromInt(100)).DivRound(total, 2).StringFixed(2))
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })
	return rows
}
