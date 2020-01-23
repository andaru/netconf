package xmlutil

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

type strPair struct{ a, b string }

func TestPrefixMap(t *testing.T) {
	for _, tc := range []struct {
		attrs     []xml.Attr
		nsTest    []strPair
		pfxTest   []strPair
		sortAttrs []xml.Attr
	}{
		// test number #00: identity check (no tests to run and an empty sortAttrs is expected)
		{},

		// #01
		{
			attrs: []xml.Attr{
				{Name: XMLName("pfx-b", "xmlns"), Value: "val-b"},
				{Name: XMLName("pfx-a", "xmlns"), Value: "val-a"},
				{Name: XMLName("pfx-c", "xmlns"), Value: "val-c"},
			},
			nsTest: []strPair{
				{a: "pfx-a", b: "val-a"},
				{a: "pfx-b", b: "val-b"},
				{a: "pfx-c", b: "val-c"},
			},
			pfxTest: []strPair{
				{b: "pfx-a", a: "val-a"},
				{b: "pfx-b", a: "val-b"},
				{b: "pfx-c", a: "val-c"},
			},
			sortAttrs: []xml.Attr{
				{Name: XMLName("pfx-a", "xmlns"), Value: "val-a"},
				{Name: XMLName("pfx-b", "xmlns"), Value: "val-b"},
				{Name: XMLName("pfx-c", "xmlns"), Value: "val-c"},
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			a := assert.New(t)
			pmap := NewPrefixMap(tc.attrs...)
			for _, tt := range tc.nsTest {
				a.Equal(tt.b, pmap.Namespace(tt.a))
			}
			for _, tt := range tc.pfxTest {
				var pfx string
				if pfxes := pmap.Prefix(tt.a); pfxes != nil {
					pfx = pfxes[0]
				}
				a.Equal(tt.b, pfx)
			}
			a.Equal(tc.sortAttrs, pmap.Attr())
		})
	}
}
