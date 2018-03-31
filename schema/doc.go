// Copyright 2018 Andrew Fort

// Package schema provides the NETCONF XML schema object and
// corresponding parser.
//
// Schema objects are constructed in terms of received XML tokens. The
// first XML token read from the input causes the schema's root node
// children to be scanned for a child schema node matching the
// received XML token.  Matches can be simple or more elaborate.
//
// There exist different schema objects for parsing NETCONF client
// sessions versus server sessions.  A NETCONF parser is constructed
// with a schema object and a standard-library XML decoder to read
// tokens from.
//
// The parser allows registration of callbacks for parser events which
// facilitate processing of a NETCONF XML session, as triggered by
// schema node events upon the client or server session schema object.
//
// Schema processing
//
// Each call to the ParseSchemaXML function made by the state machine
// will consume a single XML token from the input reader. If an error
// occurs reading the token, the state machine will exit and errors
// other than EOF will be set on the Error struct field of the parser
// state machine.
//
// If there is no error, depending on the XML token type received the
// following occurs;
//
//   xml.StartElement
//       If a matching child node is found by XML name, the tokenize
//       callback is run. The parser will change the context node to
//       the matched node and the next state will consume another
//       token.
//
//   xml.CharData
//       If a matching child node is found by type (NodeTypeText),
//       its callback is run. The parser will not change the context
//       node (as CharData tokens have no element children), the
//       will next consume another token.
//
//   xml.EndElement
//       If a matching child node is found by XML name, the
//       end-element callback is run. The parser will change the
//       context node to the parent element schema node and will next
//       consume another token.
//
//
// NETCONF Session Callbacks
//
// Parser callbacks (described above) are registered using
// ParserOption, including WithRemoteCapabilities.
package schema
