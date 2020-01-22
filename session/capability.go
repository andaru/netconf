package session

// Capabilities is a slice of strings denoting NETCONF capability URIs
type Capabilities []string

// Has returns true if uri is in the capabilities set
func (c Capabilities) Has(uri string) bool {
	for _, cap := range c {
		if uri == cap {
			return true
		}
	}
	return false
}
