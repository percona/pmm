// Auto-generated from postgres/src/include/nodes/primnodes.h - DO NOT EDIT

package pg_query

/*
 * XmlExpr - various SQL/XML functions requiring special grammar productions
 *
 * 'name' carries the "NAME foo" argument (already XML-escaped).
 * 'named_args' and 'arg_names' represent an xml_attribute list.
 * 'args' carries all other arguments.
 *
 * Note: result type/typmod/collation are not stored, but can be deduced
 * from the XmlExprOp.  The type/typmod fields are just used for display
 * purposes, and are NOT necessarily the true result type of the node.
 */
type XmlExprOp uint

const (
	IS_XMLCONCAT    XmlExprOp = iota /* XMLCONCAT(args) */
	IS_XMLELEMENT                    /* XMLELEMENT(name, xml_attributes, args) */
	IS_XMLFOREST                     /* XMLFOREST(xml_attributes) */
	IS_XMLPARSE                      /* XMLPARSE(text, is_doc, preserve_ws) */
	IS_XMLPI                         /* XMLPI(name [, args]) */
	IS_XMLROOT                       /* XMLROOT(xml, version, standalone) */
	IS_XMLSERIALIZE                  /* XMLSERIALIZE(is_document, xmlval) */
	IS_DOCUMENT                      /* xmlval IS DOCUMENT */
)
