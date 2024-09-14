// Just a wrapper for Decimals, doesn't actually tie to an implementation

package data

import (
	"math/big"
	"strconv"
)

type Decimal struct {
	repr string
}

type Integer Decimal

func NewDecimal(s string) *Decimal {
	return &Decimal{
		repr: s,
	}
}

func (n Decimal) MarshalJSON() ([]byte, error) {
	return []byte(n.repr), nil
}

func (n *Decimal) UnmarshalJSON(b []byte) error {
	if n == nil {
		*n = Decimal{}
	}
	n.repr = string(b)
	return nil
}

func (n *Decimal) String() string {
	return n.repr
}

func (n *Decimal) AsInt() int {
	return int(n.AsInt64())
}

func (n *Decimal) AsInt64() int64 {
	f, err := strconv.ParseInt(n.repr, 10, 64)
	if err == nil {
		return f
	}
	return 0
}

func (n *Decimal) AsFloat64() float64 {
	f, err := strconv.ParseFloat(n.repr, 64)
	if err == nil {
		return f
	}
	return 0
}

func (n *Decimal) AsBigInt() *big.Int {
	if i, ok := new(big.Int).SetString(n.repr, 10); ok {
		return i
	}
	return nil
}

const DecimalPrecision = uint(250)

func (n *Decimal) AsBigFloat() *big.Float {
	f, _, err := big.ParseFloat(n.repr, 10, DecimalPrecision, big.ToNearestEven)
	if err != nil {
		return f
	}
	return nil
}
