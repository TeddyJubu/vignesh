package receptionist

import "unicode"

// DetectLanguage returns a short code (en, bn, ar) from message text.
func DetectLanguage(text string) string {
	var bn, ar, latin int
	for _, r := range text {
		switch {
		case r >= 0x0980 && r <= 0x09FF:
			bn++
		case r >= 0x0600 && r <= 0x06FF:
			ar++
		case unicode.IsLetter(r) && r < 128:
			latin++
		}
	}
	if bn > latin && bn >= ar {
		return "bn"
	}
	if ar > latin && ar >= bn {
		return "ar"
	}
	return "en"
}

func languagePromptLine(code string) string {
	switch code {
	case "bn":
		return "The lead writes in Bengali. Reply in Bengali unless they switch to English."
	case "ar":
		return "The lead writes in Arabic. Reply in Arabic unless they switch to English."
	default:
		return "The lead writes in English. Reply in English unless they switch language."
	}
}
