package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

//Some kind of proof of concept for something?
func Treenode(w http.ResponseWriter, r *http.Request) {
	var parentNode, childNode1, childNode2, childNode3, childNode4, childNode5 TreeNode

	// metadata for the nodes

	parentNode.Identifier = "urn:cite2:treenodes:greekLit.tlg0011.tlg003.perseus-grc1.treesentences:1.root"
	parentNode.SentenceID = "urn:cts:greekLit:tlg0011.tlg003.perseus-grc1.treesentences:1"
	parentNode.WordToken.Text = "#root"

	childNode1.Identifier = "urn:cite2:treenodes:greekLit.tlg0011.tlg003.perseus-grc1.treesentences:1.8"
	childNode1.SentenceID = "urn:cts:greekLit:tlg0011.tlg003.perseus-grc1.treesentences:1"
	childNode1.WordToken.Text = "δέδορκά"

	childNode2.Identifier = "urn:cite2:treenodes:greekLit.tlg0011.tlg003.perseus-grc1.treesentences:1.2"
	childNode2.SentenceID = "urn:cts:greekLit:tlg0011.tlg003.perseus-grc1.treesentences:1"
	childNode2.WordToken.Text = "μέν"

	childNode3.Identifier = "urn:cite2:treenodes:greekLit.tlg0011.tlg003.perseus-grc1.treesentences:1.1"
	childNode3.SentenceID = "urn:cts:greekLit:tlg0011.tlg003.perseus-grc1.treesentences:1"
	childNode3.WordToken.Text = "ἀεὶ"

	childNode4.Identifier = "urn:cite2:treenodes:greekLit.tlg0011.tlg003.perseus-grc1.treesentences:1.5"
	childNode4.SentenceID = "urn:cts:greekLit:tlg0011.tlg003.perseus-grc1.treesentences:1"
	childNode4.WordToken.Text = "παῖ"

	childNode5.Identifier = "urn:cite2:treenodes:greekLit.tlg0011.tlg003.perseus-grc1.treesentences:1.4"
	childNode5.SentenceID = "urn:cts:greekLit:tlg0011.tlg003.perseus-grc1.treesentences:1"
	childNode5.WordToken.Text = "ὦ"

	// build the tree

	childNode4.Children = []TreeNode{childNode5}
	childNode1.Children = []TreeNode{childNode2, childNode3, childNode4}
	parentNode.Children = []TreeNode{childNode1}

	// produce JSON

	theJson, _ := json.Marshal(parentNode)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(theJson))
}
