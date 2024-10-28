package util

// Constants for all supported currencies
const (
	USD = "USD"
	EUR = "EUR"
	AUD = "AUD"
	CAD = "CAD"
)

// IsSupportedCurrency returns true if the currency is supported
func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, AUD, CAD:
		return true
	}
	return false
}
