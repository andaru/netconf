package ncerr

import (
	"fmt"
	"testing"

	"encoding/json"
	"encoding/xml"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	for _, tc := range []struct {
		err *Error

		error string
		xml   string
		json  string
	}{
		{
			err:   InUse(WithMessage("foo")),
			error: "application error tag:in-use foo",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>in-use</error-tag><error-severity>error</error-severity><error-message>foo</error-message></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"in-use\",\"error-severity\":\"error\",\"error-message\":\"foo\"}",
		},

		{
			err:   InvalidValue(WithMessage("123")),
			error: "application error tag:invalid-value 123",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>invalid-value</error-tag><error-severity>error</error-severity><error-message>123</error-message></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"invalid-value\",\"error-severity\":\"error\",\"error-message\":\"123\"}",
		},

		{
			err:   TooBig(),
			error: "application error tag:too-big",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>too-big</error-tag><error-severity>error</error-severity></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"too-big\",\"error-severity\":\"error\"}",
		},

		{
			err:   MissingAttribute("attrib", "elem"),
			error: "application error tag:missing-attribute bad-attribute:attrib bad-element:elem",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>missing-attribute</error-tag><error-severity>error</error-severity><error-info><bad-attribute>attrib</bad-attribute><bad-element>elem</bad-element></error-info></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"missing-attribute\",\"error-severity\":\"error\",\"error-info\":{\"bad-attribute\":\"attrib\",\"bad-element\":\"elem\"}}",
		},

		{
			err:   BadAttribute("attrib", "elem"),
			error: "application error tag:bad-attribute bad-attribute:attrib bad-element:elem",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>bad-attribute</error-tag><error-severity>error</error-severity><error-info><bad-attribute>attrib</bad-attribute><bad-element>elem</bad-element></error-info></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"bad-attribute\",\"error-severity\":\"error\",\"error-info\":{\"bad-attribute\":\"attrib\",\"bad-element\":\"elem\"}}",
		},

		{
			err:   UnknownAttribute("attrib", "elem"),
			error: "application error tag:unknown-attribute bad-attribute:attrib bad-element:elem",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>unknown-attribute</error-tag><error-severity>error</error-severity><error-info><bad-attribute>attrib</bad-attribute><bad-element>elem</bad-element></error-info></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"unknown-attribute\",\"error-severity\":\"error\",\"error-info\":{\"bad-attribute\":\"attrib\",\"bad-element\":\"elem\"}}",
		},

		{
			err:   MissingElement("elem"),
			error: "application error tag:missing-element bad-element:elem",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>missing-element</error-tag><error-severity>error</error-severity><error-info><bad-element>elem</bad-element></error-info></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"missing-element\",\"error-severity\":\"error\",\"error-info\":{\"bad-element\":\"elem\"}}",
		},

		{
			err:   BadElement("elem"),
			error: "application error tag:bad-element bad-element:elem",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>bad-element</error-tag><error-severity>error</error-severity><error-info><bad-element>elem</bad-element></error-info></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"bad-element\",\"error-severity\":\"error\",\"error-info\":{\"bad-element\":\"elem\"}}",
		},

		{
			err:   UnknownElement("elem"),
			error: "application error tag:unknown-element bad-element:elem",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>application</error-type><error-tag>unknown-element</error-tag><error-severity>error</error-severity><error-info><bad-element>elem</bad-element></error-info></rpc-error>",
			json:  "{\"error-type\":\"application\",\"error-tag\":\"unknown-element\",\"error-severity\":\"error\",\"error-info\":{\"bad-element\":\"elem\"}}",
		},

		{
			err:   LockDenied("session-id-123", WithMessage("foo")),
			error: "protocol error tag:lock-denied session-id:session-id-123 foo",
			xml:   "<rpc-error xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\"><error-type>protocol</error-type><error-tag>lock-denied</error-tag><error-severity>error</error-severity><error-message>foo</error-message><error-info><session-id>session-id-123</session-id></error-info></rpc-error>",
			json:  "{\"error-type\":\"protocol\",\"error-tag\":\"lock-denied\",\"error-severity\":\"error\",\"error-message\":\"foo\",\"error-info\":{\"session-id\":\"session-id-123\"}}",
		},
	} {
		t.Run(fmt.Sprintf("%v", tc.err), func(t *testing.T) {
			// confirm basic marshaling works for XML and JSON
			check := assert.New(t)
			bXML, _ := xml.Marshal(tc.err)
			bJSON, _ := json.Marshal(tc.err)
			check.Equal(tc.error, tc.err.Error())
			check.Equal(tc.json, string(bJSON))
			check.Equal(tc.xml, string(bXML))

			// confirm unmarshaling works for both XML and JSON by
			// unmarshaling the marshaled error text and then
			// marshaling the new object and comparing to the original
			// expected output.
			ev := Error{}
			if check.NoError(xml.Unmarshal(bXML, &ev)) {
				evXML, _ := xml.Marshal(ev)
				check.Equal(tc.xml, string(evXML))
			}
			ev = Error{}
			if check.NoError(json.Unmarshal(bJSON, &ev)) {
				evJSON, _ := json.Marshal(ev)
				check.Equal(tc.json, string(evJSON))
			}
		})
	}
}
