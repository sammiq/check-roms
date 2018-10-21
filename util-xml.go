package main

import (
	"github.com/antchfx/xmlquery"
)

//mapAttr returns the name and value of every attribute of a node
func mapAttr(node *xmlquery.Node) map[string]string {
	attrMap := make(map[string]string)
	for _, attr := range node.Attr {
		attrMap[attr.Name.Local] = attr.Value
	}
	return attrMap
}

//findAttr returns the value of an attribute of a node, or an empty string if not found
func findAttr(node *xmlquery.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

//NodeSet is an alias a map of nodes without values as a node set
type NodeSet = map[*xmlquery.Node]struct{}

//childNodeSet generates a set containing the direct child nodes of the passed in node
//with the matching element type
func childNodeSet(parentNode *xmlquery.Node, elementType string) NodeSet {
	roms := make(NodeSet)
	for other := parentNode.FirstChild; other != nil; other = other.NextSibling {
		if other.Type == xmlquery.ElementNode && other.Data == elementType {
			roms[other] = struct{}{}
		}
	}
	return roms
}
