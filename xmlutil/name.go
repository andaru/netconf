package xmlutil

import "encoding/xml"

// XMLName is a shortcut for creating xml.Name, where typically you want at least
// a local name, and perhaps a namespace value as well.
func XMLName(local string, spaces ...string) xml.Name {
	n := xml.Name{Local: local}
	if len(spaces) > 0 {
		n.Space = spaces[0]
	}
	return n
}
