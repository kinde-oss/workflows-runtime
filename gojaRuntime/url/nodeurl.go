package url

import (
	"net/url"
	"strings"
)

type SearchParam struct {
	Name  string
	Value string
}

func (sp *SearchParam) Encode() string {
	return sp.string(true)
}

func escapeSearchParam(s string) string {
	return escape(s, &tblEscapeURLQueryParam, true)
}

func (sp *SearchParam) string(encode bool) string {
	if encode {
		return escapeSearchParam(sp.Name) + "=" + escapeSearchParam(sp.Value)
	} else {
		return sp.Name + "=" + sp.Value
	}
}

type SearchParams []SearchParam

func (s SearchParams) Len() int {
	return len(s)
}

func (s SearchParams) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SearchParams) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) < 0
}

func (s SearchParams) Encode() string {
	var sb strings.Builder
	for i, v := range s {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(v.Encode())
	}
	return sb.String()
}

func (s SearchParams) String() string {
	var sb strings.Builder
	for i, v := range s {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(v.string(false))
	}
	return sb.String()
}

type nodeURL struct {
	Url          *url.URL
	SearchParams SearchParams
}

type UrlSearchParams nodeURL

// This methods ensures that the url.URL has the proper RawQuery based on the searchParam
// structs. If a change is made to the searchParams we need to keep them in sync.
func (nu *nodeURL) syncSearchParams() {
	if nu.rawQueryUpdateNeeded() {
		nu.Url.RawQuery = nu.SearchParams.Encode()
	}
}

func (nu *nodeURL) rawQueryUpdateNeeded() bool {
	return len(nu.SearchParams) > 0 && nu.Url.RawQuery == ""
}

func (nu *nodeURL) String() string {
	return nu.Url.String()
}

func (sp *UrlSearchParams) hasName(name string) bool {
	for _, v := range sp.SearchParams {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (sp *UrlSearchParams) hasValue(name, value string) bool {
	for _, v := range sp.SearchParams {
		if v.Name == name && v.Value == value {
			return true
		}
	}
	return false
}

func (sp *UrlSearchParams) getValues(name string) []string {
	vals := make([]string, 0, len(sp.SearchParams))
	for _, v := range sp.SearchParams {
		if v.Name == name {
			vals = append(vals, v.Value)
		}
	}

	return vals
}

func (sp *UrlSearchParams) getFirstValue(name string) (string, bool) {
	for _, v := range sp.SearchParams {
		if v.Name == name {
			return v.Value, true
		}
	}

	return "", false
}

func parseSearchQuery(query string) (ret SearchParams) {
	if query == "" {
		return
	}

	query = strings.TrimPrefix(query, "?")

	for _, v := range strings.Split(query, "&") {
		if v == "" {
			continue
		}
		pair := strings.SplitN(v, "=", 2)
		l := len(pair)
		if l == 1 {
			ret = append(ret, SearchParam{Name: unescapeSearchParam(pair[0]), Value: ""})
		} else if l == 2 {
			ret = append(ret, SearchParam{Name: unescapeSearchParam(pair[0]), Value: unescapeSearchParam(pair[1])})
		}
	}

	return
}
