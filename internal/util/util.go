package util

import "regexp"

var TrailingComma = regexp.MustCompile(`,(\s*[}\]])`)

func StripJSONC(data []byte) []byte {
	var out []byte
	inString := false
	inBlockComment := false
	inLineComment := false
	escaped := false

	for i := 0; i < len(data); i++ {
		b := data[i]

		if inLineComment {
			if b == '\n' {
				inLineComment = false
				out = append(out, b)
			}
			continue
		}

		if inBlockComment {
			if b == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				inString = false
			}
			out = append(out, b)
			continue
		}

		if b == '/' && i+1 < len(data) {
			if data[i+1] == '/' {
				inLineComment = true
				i++
				continue
			}
			if data[i+1] == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		if b == '"' {
			inString = true
		}

		out = append(out, b)
	}

	s := string(out)
	s = TrailingComma.ReplaceAllString(s, "$1")
	return []byte(s)
}
