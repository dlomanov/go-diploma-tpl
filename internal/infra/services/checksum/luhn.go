package checksum

func ValidateLuhn(s []byte) bool {
	var num, sum int

	parity := len(s) % 2

	for i, v := range s {
		if v < '0' || v > '9' {
			return false
		}

		num = int(v) - '0'

		if i%2 != parity {
			sum += num
			continue
		}

		num *= 2
		if num > 9 {
			num -= 9
		}
		sum += num
	}

	return sum%10 == 0
}
