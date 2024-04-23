package tokens

const (
	// minLengthForTickerName represents the minimum number of characters a token's ticker can have
	minLengthForTickerName = 3
	// maxLengthForTickerName represents the maximum number of characters a token's ticker can have
	maxLengthForTickerName = 10
	// maxLengthESDTPrefix represents the maximum number of characters a token's prefix can have
	maxLengthESDTPrefix = 4
	// minLengthESDTPrefix represents the minimum number of characters a token's prefix can have
	minLengthESDTPrefix = 1
)

// IsValidTokenPrefix checks if the token prefix is valid
func IsValidTokenPrefix(prefix string) bool {
	prefixLen := len(prefix)
	if prefixLen > maxLengthESDTPrefix || prefixLen < minLengthESDTPrefix {
		return false
	}

	for _, ch := range prefix {
		isLowerCaseCharacter := ch >= 'a' && ch <= 'z'
		isNumber := ch >= '0' && ch <= '9'
		isAllowedChar := isLowerCaseCharacter || isNumber
		if !isAllowedChar {
			return false
		}
	}

	return true
}

// IsTickerValid checks if the token ticker is valid
func IsTickerValid(ticker string) bool {
	if !IsTokenTickerLenCorrect(len(ticker)) {
		return false
	}

	for _, ch := range ticker {
		isUpperCaseCharacter := ch >= 'A' && ch <= 'Z'
		isNumber := ch >= '0' && ch <= '9'
		isAllowedChar := isUpperCaseCharacter || isNumber
		if !isAllowedChar {
			return false
		}
	}

	return true
}

// IsTokenTickerLenCorrect checks if the token ticker len is correct
func IsTokenTickerLenCorrect(tokenTickerLen int) bool {
	return !(tokenTickerLen < minLengthForTickerName || tokenTickerLen > maxLengthForTickerName)
}
