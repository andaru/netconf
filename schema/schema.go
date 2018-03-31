package schema

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/andaru/netconf/ncerr"
	"github.com/andaru/xmlstream"
	"github.com/pkg/errors"
)

// ParserEvent is a schema parsing event
type ParserEvent struct {
	Callback func()
	Err      error
}

func ValidateSchema(ctx context.Context, n *xmlstream.Node, es EventSender) {
	if n != nil {
		n.Iter(childValidate(ctx, es))
	}
}

func childValidate(ctx context.Context, es EventSender) xmlstream.NodeIterFn {
	return func(it *xmlstream.Node) error {
		if it.Validator != nil {
			it.Validator(ctx, it, it.Status.SE)
		}
		if it.T == xmlstream.NodeTypeElement {
			return it.Iter(childValidate(ctx, es))
		}
		return nil
	}
}

// ParseSchemaXML returns a state machine function which consumes the
// next XML token from d given the context schema node n. events is
// the channel to receive events such as parsing errors and must be
// consumed concurrently.
func ParseSchemaXML(n *xmlstream.Node, d *xml.Decoder, es EventSender) xmlstream.StateFn {
	if n == nil {
		return nil
	}

	return func(ctx context.Context, sm *xmlstream.StateMachine) xmlstream.StateFn {
		// check for context cancellation before d.Token() blocks.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return xmlstream.BailWithError(ctxErr)
		}

		token, err := d.Token()
		if err != nil {
			if err != io.EOF {
				es.CriticalError(ctx, ncerr.MalformedMessage(ncerr.WithMessage(err.Error())))
			}
			return xmlstream.BailWithError(err)
		}
		token = xml.CopyToken(token)
		switch token := token.(type) {

		case xml.StartElement:
			var choice *xmlstream.Node
			n.Iter(func(it *xmlstream.Node) error {
				if it.T != xmlstream.NodeTypeElement {
					return nil
				}

				if it.Name == nameWildcard {
					// wildcard match; use it, but continue iterating
					// over other children for an explicit match.
					choice = it
				} else if it.Name == token.Name {
					// explicit match
					choice = it
					return errStopIteration
				}

				return nil
			})

			if choice != nil {
				runCallbacks(ctx, choice, token, xmlstream.CBStartElement, xmlstream.CBSEOccurs)
				return ParseSchemaXML(choice, d, es)
			}
			es.Error(ctx, errNodeUnknownElement(n, token))
			// events <- &ParserEvent{Err: }

		case xml.CharData:
			if token = xml.CharData(bytes.TrimSpace(token.Copy())); len(token) > 0 {
				if n.Iter(func(it *xmlstream.Node) error {
					if it.T != xmlstream.NodeTypeText {
						return nil
					}
					runCallbacks(ctx, it, token, xmlstream.CBText)
					return errStopIteration
				}) == nil {
					// events <- &ParserEvent{Err: errNodeUnexpectedCData(n, token)}
					es.Error(ctx, errNodeUnexpectedCData(n, token))
				}
			}

		case xml.EndElement:
			runCallbacks(ctx, n, token, xmlstream.CBEndElement)
			// consume the next token in the context of the parent
			// element schema node
			return ParseSchemaXML(n.ParentElement(), d, es)

		case xml.ProcInst:
			n.Iter(func(it *xmlstream.Node) error {
				if it.T != xmlstream.NodeTypeProcInst {
					return nil
				}
				procInst, ok := it.Value.(xml.ProcInst)
				if !ok {
					panic(errors.Errorf("error: want xml.ProcInst, got %T", it.Value))
				}
				if procInst.Target == token.Target {
					runCallbacks(ctx, it, token, xmlstream.CBStartElement)
					return errStopIteration
				}
				return nil
			})

		case xml.Comment, xml.Directive:
			// ignore comments and directives

		default:
			// note that we covered all types mentioned for xml.Token
			// (interface{}) in the encoding/xml package (go 1.10)
			// above.
		}

		return ParseSchemaXML(n, d, es)
	}

}

func runCallbacks(ctx context.Context, n *xmlstream.Node, t xml.Token, ns ...xml.Name) {
	if n != nil && len(ns) > 0 {
		n.Iter(runCB(ctx, t, ns))
	}
}

func runCB(ctx context.Context, token xml.Token, names []xml.Name) xmlstream.NodeIterFn {
	return func(it *xmlstream.Node) error {
		if it.T != xmlstream.NodeTypeCB {
			return nil
		}
		for _, name := range names {
			if it.Name != name {
				continue
			}
			switch cb := it.Value.(type) {
			case xmlstream.NodeIterFn:
				cb(it)
			case xmlstream.NodeTokenCallback:
				cb(ctx, it, token)
			default:
				panic(fmt.Errorf("unexpected callback for node %#v type %T: %#v",
					it.ParentElement(), it.Value, it.Value))
			}
		}
		return nil
	}
}

func elemStr(n xml.Name) string { return genElemStr(n, "<") }

func pprintElem(t xml.Token) string {
	switch t := t.(type) {
	case xml.StartElement:
		return genElemStr(t.Name, "<")
	case xml.EndElement:
		return genElemStr(t.Name, "</")
	}
	return ""
}

func genElemStr(n xml.Name, pfx string) string {
	local := n.Local
	if local == "" {
		return ""
	}
	if ns := n.Space; ns != "" {
		return pfx + local + ` xmlns="` + ns + `">`
	}
	return pfx + local + ">"
}

func errNodeUnexpectedCData(n *xmlstream.Node, cdata xml.CharData) error {
	return errors.WithStack(ncerr.UnknownElement("CDATA", ncerr.WithMessage(fmt.Sprintf(
		"unexpected character data found in element %s: %q", elemStr(n.Name), bytes.TrimSpace(cdata)))))
}

func errNodeUnknownElement(n *xmlstream.Node, se xml.StartElement) error {
	return errors.WithStack(ncerr.UnknownElement(se.Name.Local, ncerr.WithMessage(fmt.Sprintf(
		"unexpected element %s found in element %s", elemStr(se.Name), elemStr(n.Name)))))
}

var errStopIteration = errors.New("stop iteration")
