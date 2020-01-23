package xmlutil

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXMLName(t *testing.T) {
	for _, tc := range []struct {
		local  string
		spaces []string
		want   xml.Name
	}{
		{local: "foo", want: xml.Name{Local: "foo"}},
		{local: "foo", spaces: []string{"bar"}, want: xml.Name{Local: "foo", Space: "bar"}},
		{local: "foo", spaces: []string{"bar", "baz"}, want: xml.Name{Local: "foo", Space: "bar"}},
		{want: xml.Name{}},
	} {
		t.Run(fmt.Sprintf("%v", tc.want), func(t *testing.T) { assert.New(t).Equal(tc.want, XMLName(tc.local, tc.spaces...)) })
	}
}
