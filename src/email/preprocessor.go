package email

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func preprocessEmailHTML(html []byte) ([]byte, error) {
	var errs []error

	rules := make(cssRules)
	remaining := html
	out := make([]byte, 0, len(html))
nextTag:
	for {
		locs := reOpeningTag.FindSubmatchIndex(remaining)
		if locs == nil {
			out = append(out, remaining...)
			break
		}
		beforeTag := remaining[:locs[0]]
		// fullMatch := remaining[locs[0]:locs[1]]
		tagName := remaining[locs[2]:locs[3]]
		var atts []byte
		if locs[4] != -1 {
			atts = remaining[locs[4]:locs[5]]
		}
		selfClose := remaining[locs[6]:locs[7]]

		// Advance to just past the end of the tag. Sub-components of the tag will be
		// parsed separately.
		remaining = remaining[min(locs[1], len(remaining)):]

		out = append(out, beforeTag...)
		if string(tagName) == "style" {
			// Naively find the closing tag, then parse the CSS rules.
			locs := reClosingStyleTag.FindIndex(remaining)
			if locs == nil {
				// No closing tag? Strange. Just continue to the next iteration.
				continue nextTag
			}
			rawCSS := remaining[:locs[0]]
			remaining = remaining[min(locs[1], len(remaining)):] // Advance to past </style>

			err := parseCSSRules(rawCSS, rules)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to parse CSS: %w", err))
			}
		} else {
			// Process the tag's attributes, replacing `style` and `class` with a single new
			// style attribute.

			out = append(out, '<')
			out = append(out, tagName...)

			// First find any styles pertaining to the tag alone.
			var finalStyles []string
			if styles, ok := rules[string(tagName)]; ok {
				finalStyles = append(finalStyles, styles...)
			}

			// Parse / output attributes and gather up styles from classes and styles.
			attsRemaining := atts
			for {
				locs := reAttribute.FindSubmatchIndex(attsRemaining)
				if locs == nil {
					break
				}
				fullMatch := attsRemaining[locs[0]:locs[1]]
				attName := attsRemaining[locs[2]:locs[3]]
				attValue := attsRemaining[locs[4]:locs[5]]

				attsRemaining = attsRemaining[min(locs[1], len(attsRemaining)):] // Advance to next attribute

				switch string(attName) {
				case "class":
					// Classes are parsed and their styles pulled out of the map.
					classesRemaining := attValue
					for {
						classesRemaining = bytes.TrimLeft(classesRemaining, " ")
						if len(classesRemaining) == 0 {
							break
						}
						afterClass := bytes.IndexByte(classesRemaining, ' ')
						if afterClass == -1 {
							afterClass = len(classesRemaining)
						}
						class := string(classesRemaining[:afterClass])
						classesRemaining = classesRemaining[afterClass:]

						if styles, ok := rules["."+class]; ok {
							finalStyles = append(finalStyles, styles...)
						} else {
							errs = append(errs, fmt.Errorf("unknown class %s", class))
						}
					}
				case "style":
					// Existing inline styles are just dumped in as is (minus semicolons).
					finalStyles = append(finalStyles, string(bytes.Trim(attValue, ";")))
				default:
					// All other attributes are immediately echoed.
					out = append(out, ' ')
					out = append(out, fullMatch...)
				}
			}

			// Emit final styles and close the tag.
			if len(finalStyles) > 0 {
				out = append(out, []byte(` style="`)...)
				out = append(out, []byte(strings.Join(finalStyles, ";"))...)
				out = append(out, []byte(`;"`)...)
			}

			out = append(out, selfClose...)
			out = append(out, '>')
		}
	}

	return out, errors.Join(errs...)
}

var reOpeningTag = regexp.MustCompile(`<([a-zA-Z-]+)((?:\s+[a-zA-Z-]+=".*?")+)?\s*(/?)>`)
var reClosingStyleTag = regexp.MustCompile(`</style\s*>`)
var reAttribute = regexp.MustCompile(`([a-zA-Z-]+)="(.*?)"`)
var reCSSRule = regexp.MustCompile(`(?s)(\.?[a-zA-Z0-9-]+)\s\{(.*?)}`)
var reCSSComment = regexp.MustCompile(`/\*.*?\*/`)
var reCSSStyle = regexp.MustCompile(`([a-zA-Z0-9-]+):\s*(.*?);`)

type cssRules map[string][]string

func parseCSSRules(css []byte, rules cssRules) error {
	// We are a bit lazier with the parsing here.
	cssMinusComments := reCSSComment.ReplaceAllLiteralString(string(css), "")
	for _, rule := range reCSSRule.FindAllStringSubmatch(cssMinusComments, -1) {
		selector := rule[1]
		styles := rule[2]

		var mapStyles []string
		for line := range strings.SplitSeq(styles, "\n") {
			match := reCSSStyle.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			mapStyles = append(mapStyles, fmt.Sprintf("%s:%s", match[1], match[2]))
		}
		rules[selector] = mapStyles
	}

	return nil
}
