package currency

// eur represents the Euro.
type eur struct{ currency }

// Format returns the amount with € suffix: "12.50 €".
func (e eur) Format(amount string) string {
	return amount + " " + e.symbol
}

// EUR is the Euro currency.
var EUR Currency = eur{currency: mustBase("EUR")}
