package xmlutil

import (
	"encoding/xml"
	"sort"
)

// PrefixMap is a prefix to namespce URI map
type PrefixMap map[string]string

// NewPrefixMap returns a PrefixMap, containing the passed XML attributes
func NewPrefixMap(attrs ...xml.Attr) PrefixMap {
	pmap := PrefixMap{}
	for _, attr := range attrs {
		if attr.Name.Space == "xmlns" {
			pmap[attr.Name.Local] = attr.Value
		}
	}
	return pmap
}

// Attr returns the prefix map contents as a series of xmlns:<prefix>=<nsuri> attributes,
// sorted lexically by prefix.
func (m PrefixMap) Attr() (a []xml.Attr) {
	for k, v := range m {
		a = append(a, xml.Attr{Name: xml.Name{Space: "xmlns", Local: k}, Value: v})
	}
	if len(a) > 0 {
		// sort lexically by prefix
		sort.Slice(a, func(i int, j int) bool { return a[i].Name.Local < a[j].Name.Local })
	}
	return a
}

// Namespace returns the namespace URI for the given prefix
func (m PrefixMap) Namespace(prefix string) string { return m[prefix] }

// Prefix returns any prefixes found for the namespace URI
func (m PrefixMap) Prefix(nsURI string) (pfxes []string) {
	for k, v := range m {
		if nsURI == v {
			pfxes = append(pfxes, k)
		}
	}
	return pfxes
}
