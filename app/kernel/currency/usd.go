package currency

// usd represents the US Dollar.
type usd struct{ currency }

// Format returns the amount with $ prefix: "$12.50".
func (u usd) Format(amount string) string {
	return u.symbol + amount
}

// USD is the US Dollar currency.
var USD Currency = usd{currency: mustBase("USD")}

func mustBase(code string) currency {
	c, err := baseForCode(code)
	if err != nil {
		panic(err)
	}
	return c
}
