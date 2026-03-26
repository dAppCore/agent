// SPDX-License-Identifier: EUPL-1.2

package agentic

func sanitiseBranchSlug(text string, max int) string {
	out := make([]rune, 0, len(text))
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		default:
			out = append(out, '-')
		}

		if max > 0 && len(out) >= max {
			break
		}
	}

	return trimRuneEdges(string(out), '-')
}

func sanitisePlanSlug(text string) string {
	out := make([]rune, 0, len(text))
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		case r == ' ':
			out = append(out, '-')
		}
	}

	slug := collapseRepeatedRune(string(out), '-')
	slug = trimRuneEdges(slug, '-')
	if len(slug) > 30 {
		slug = slug[:30]
	}

	return trimRuneEdges(slug, '-')
}

func sanitiseFilename(text string) string {
	out := make([]rune, 0, len(text))
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}

	return string(out)
}

func collapseRepeatedRune(text string, target rune) string {
	runes := []rune(text)
	out := make([]rune, 0, len(runes))
	lastWasTarget := false

	for _, r := range runes {
		if r == target {
			if lastWasTarget {
				continue
			}
			lastWasTarget = true
		} else {
			lastWasTarget = false
		}

		out = append(out, r)
	}

	return string(out)
}

func trimRuneEdges(text string, target rune) string {
	runes := []rune(text)
	start := 0
	end := len(runes)

	for start < end && runes[start] == target {
		start++
	}

	for end > start && runes[end-1] == target {
		end--
	}

	return string(runes[start:end])
}
