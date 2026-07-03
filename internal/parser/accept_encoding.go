package parser

import (
	"strconv"
	"strings"
)

type encodingSpec struct {
	value string
	q     float64
}

// SelectAcceptEncoding picks the most suitable encoding from supported.
// An empty string means no acceptable encoding exists (406 Not Acceptable).
func SelectAcceptEncoding(acceptValues []string, supported []string) string {
	specs := parseAcceptEncoding(acceptValues)

	best := ""
	bestQ := -1.0

	for _, offer := range supported {
		q := matchEncodingQ(specs, strings.ToLower(offer))
		if q > bestQ {
			bestQ = q
			best = offer
		}
	}

	if bestQ > 0 {
		return best
	}

	if effectiveIdentityQ(specs) > 0 {
		return "identity"
	}

	return ""
}

// parseAcceptEncoding parses Accept-Encoding header field-values.
func parseAcceptEncoding(values []string) []encodingSpec {
	var specs []encodingSpec

	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			spec := encodingSpec{q: 1.0}

			name, params, ok := strings.Cut(part, ";")
			spec.value = strings.ToLower(strings.TrimSpace(name))
			if !ok {
				specs = append(specs, spec)
				continue
			}

			for param := range strings.SplitSeq(params, ";") {
				param = strings.TrimSpace(param)
				if len(param) < 2 || !strings.EqualFold(param[:2], "q=") {
					continue
				}

				q, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64)
				if err == nil {
					spec.q = q
				}
			}

			specs = append(specs, spec)
		}
	}

	return specs
}

// matchEncodingQ returns the q-value for an encoding. An explicit spec takes
// precedence over a wildcard; -1 means the encoding is not offered.
func matchEncodingQ(specs []encodingSpec, encoding string) float64 {
	wildcardQ := -1.0
	for _, spec := range specs {
		if spec.value == encoding {
			return spec.q
		}
		if spec.value == "*" {
			wildcardQ = spec.q
		}
	}
	return wildcardQ
}

func effectiveIdentityQ(specs []encodingSpec) float64 {
	identityExplicit := false
	identityQ := 1.0
	hasWildcard := false
	wildcardQ := 1.0

	for _, spec := range specs {
		switch spec.value {
		case "identity":
			identityExplicit = true
			identityQ = spec.q
		case "*":
			hasWildcard = true
			wildcardQ = spec.q
		}
	}

	if identityExplicit {
		return identityQ
	}
	if hasWildcard {
		return wildcardQ
	}

	return 1.0
}
