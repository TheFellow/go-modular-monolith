package measurement

import "fmt"

const (
	ozToMl = 29.5735
	clToMl = 10.0
)

// Volume represents a liquid measurement stored in milliliters.
type Volume struct {
	ml float64
}

func Milliliters(v float64) Volume { return Volume{ml: v} }
func Ounces(v float64) Volume      { return Volume{ml: v * ozToMl} }
func Centiliters(v float64) Volume { return Volume{ml: v * clToMl} }

func (v Volume) Ml() float64 { return v.ml }
func (v Volume) Oz() float64 { return v.ml / ozToMl }
func (v Volume) Cl() float64 { return v.ml / clToMl }

func (v Volume) Add(other Volume) Volume { return Volume{ml: v.ml + other.ml} }
func (v Volume) Sub(other Volume) Volume { return Volume{ml: v.ml - other.ml} }
func (v Volume) Mul(n float64) Volume    { return Volume{ml: v.ml * n} }
func (v Volume) Div(n float64) Volume    { return Volume{ml: v.ml / n} }

func (v Volume) IsZero() bool               { return v.ml == 0 }
func (v Volume) LessThan(other Volume) bool { return v.ml < other.ml }

func (v Volume) String() string {
	if v.ml >= 100 {
		return fmt.Sprintf("%.0f ml", v.ml)
	}
	return fmt.Sprintf("%.1f oz", v.Oz())
}
