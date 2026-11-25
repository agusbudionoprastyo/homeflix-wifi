package utils

import (
	"path/filepath"
	"strings"
)

var subtitleLangMap = map[string]string{
	"en": "English", "eng": "English", "english": "English",
	"es": "Spanish", "spa": "Spanish", "spanish": "Spanish",
	"fr": "French", "fre": "French", "fra": "French", "french": "French",
	"de": "German", "ger": "German", "deu": "German", "german": "German",
	"it": "Italian", "ita": "Italian", "italian": "Italian",
	"pt": "Portuguese", "por": "Portuguese", "portuguese": "Portuguese",
	"ru": "Russian", "rus": "Russian", "russian": "Russian",
	"ja": "Japanese", "jpn": "Japanese", "japanese": "Japanese",
	"ko": "Korean", "kor": "Korean", "korean": "Korean",
	"zh": "Chinese", "chi": "Chinese", "zho": "Chinese", "chinese": "Chinese",
	"ar": "Arabic", "ara": "Arabic", "arabic": "Arabic",
	"hi": "Hindi", "hin": "Hindi", "hindi": "Hindi",
	"nl": "Dutch", "dut": "Dutch", "nld": "Dutch", "dutch": "Dutch",
	"sv": "Swedish", "swe": "Swedish", "swedish": "Swedish",
	"no": "Norwegian", "nor": "Norwegian", "norwegian": "Norwegian",
	"da": "Danish", "dan": "Danish", "danish": "Danish",
	"fi": "Finnish", "fin": "Finnish", "finnish": "Finnish",
	"pl": "Polish", "pol": "Polish", "polish": "Polish",
	"tr": "Turkish", "tur": "Turkish", "turkish": "Turkish",
	"he": "Hebrew", "heb": "Hebrew", "hebrew": "Hebrew",
	"th": "Thai", "tha": "Thai", "thai": "Thai",
	"vi": "Vietnamese", "vie": "Vietnamese", "vietnamese": "Vietnamese",
	"id": "Indonesian", "ind": "Indonesian", "indo": "Indonesian", "indonesian": "Indonesian", "bahasa": "Indonesian",
	"ms": "Malay", "may": "Malay", "msa": "Malay", "malay": "Malay",
	"bn": "Bengali", "ben": "Bengali", "bengali": "Bengali",
}

var subtitleModifiers = map[string]bool{
	"sdh": true, "cc": true, "forced": true, "default": true, "hi": true,
}

func splitSubtitleTokens(stem string) []string {
	return strings.FieldsFunc(strings.ToLower(stem), func(r rune) bool {
		switch r {
		case '.', '_', '-', ' ', '[', ']', '(', ')':
			return true
		default:
			return false
		}
	})
}

// ExtractSubtitleLanguageFromFilename detects a subtitle language from trailing filename tokens (e.g. movie.id.srt).
func ExtractSubtitleLanguageFromFilename(filename string) string {
	base := filepath.Base(filename)
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	tokens := splitSubtitleTokens(base)
	for i := len(tokens) - 1; i >= 0; i-- {
		tok := tokens[i]
		if subtitleModifiers[tok] {
			continue
		}
		if lang, ok := subtitleLangMap[tok]; ok {
			return lang
		}
		break
	}
	return "Unknown"
}

// StripSubtitleLanguageSuffix removes trailing language/modifier tokens from a subtitle base name.
func StripSubtitleLanguageSuffix(baseName string) string {
	trimmed := baseName
	for {
		idx := strings.LastIndexAny(trimmed, "._-")
		if idx <= 0 || idx >= len(trimmed)-1 {
			break
		}
		tail := strings.ToLower(trimmed[idx+1:])
		if subtitleModifiers[tail] {
			trimmed = trimmed[:idx]
			continue
		}
		if _, ok := subtitleLangMap[tail]; ok {
			trimmed = trimmed[:idx]
			continue
		}
		break
	}
	return trimmed
}
