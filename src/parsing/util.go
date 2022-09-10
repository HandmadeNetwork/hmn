package parsing

import "regexp"

func extract(re *regexp.Regexp, src []byte, subexpName string) []byte {
	m := re.FindSubmatch(src)
	if m == nil {
		return nil
	}
	return m[re.SubexpIndex(subexpName)]
}

func extractMap(re *regexp.Regexp, src []byte) map[string][]byte {
	m := re.FindSubmatch(src)
	if m == nil {
		return nil
	}
	res := make(map[string][]byte)
	for _, name := range re.SubexpNames() {
		if name != "" {
			i := re.SubexpIndex(name)
			res[name] = m[i]
		}
	}
	res["all"] = m[0]
	return res
}

func extractAll(re *regexp.Regexp, src []byte, subexpName string) [][]byte {
	m := re.FindAllSubmatch(src, -1)
	if m == nil {
		return nil
	}
	return m[re.SubexpIndex(subexpName)]
}

func extractAllMap(re *regexp.Regexp, src []byte) map[string][][]byte {
	m := re.FindAllSubmatch(src, -1)
	if m == nil {
		return nil
	}
	res := make(map[string][][]byte)
	for i, name := range re.SubexpNames() {
		if name != "" {
			var vals [][]byte
			for _, specificMatch := range m {
				vals = append(vals, specificMatch[i])
			}
			res[name] = vals
		}
	}
	res["all"] = m[0]
	return res
}
