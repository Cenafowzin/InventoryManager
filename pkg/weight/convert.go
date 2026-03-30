package weight

const lbsToKg = 0.453592

func ToKg(value float64, unit string) float64 {
	if unit == "lbs" {
		return value * lbsToKg
	}
	return value
}
