package helpers

func LuhnCheck(ordernumber string) bool {

	var sum int
	var digit int
	var even = false

	// iterate over digits from right to left
	for i := len(ordernumber) - 1; i >= 0; i-- {
		digit = int(ordernumber[i] - '0')
		if even {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit

		even = !even
	}

	// return true if sum is divisible by 10
	return sum%10 == 0
}
