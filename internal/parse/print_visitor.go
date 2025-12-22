package parse

import (
	"github.com/armchr/codeapi/internal/model/ast"
	"context"
	"fmt"
	"os"
	"strconv"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type PrintVisitor struct {
	translate *TranslateFromSyntaxTree
	//logger    *zap.Logger
	indent  int
	content string
}

func NewPrintVisitor(ts *TranslateFromSyntaxTree) *PrintVisitor {
	return &PrintVisitor{
		translate: ts,
		//logger:    logger,
		indent:  0,
		content: "",
	}
}

func PrintSyntaxTree(ctx context.Context, tsNode *tree_sitter.Node, content []byte) string {
	ts := &TranslateFromSyntaxTree{
		FileContent: content,
	}
	pv := NewPrintVisitor(ts)
	ts.Visitor = pv
	pv.TraverseNode(ctx, tsNode, ast.InvalidNodeID)
	return pv.content
}

func (pv *PrintVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode == nil {
		return ast.InvalidNodeID
	}

	for i := 0; i < pv.indent; i++ {
		pv.content += "  "
	}

	rangeStr := fmt.Sprintf("[%d:%d - %d:%d]", tsNode.StartPosition().Row, tsNode.StartPosition().Column, tsNode.EndPosition().Row, tsNode.EndPosition().Column)

	pv.content += fmt.Sprintf("kind=%s, named=%s, name=%s, range=%s\n",
		tsNode.Kind(), strconv.FormatBool(tsNode.IsNamed()), pv.translate.GetTreeNodeName(tsNode), rangeStr,
	)

	pv.indent += 2

	pv.translate.TraverseChildren(ctx, tsNode, scopeID)

	pv.indent -= 2

	/*
		if pv.indent == 0 {
			//pv.logger.Info("Syntax Tree:\n" + pv.content)
			pv.WriteToFile("syntax_tree.txt")
			pv.content = ""
		}
	*/

	return ast.InvalidNodeID
}

func (pv *PrintVisitor) WriteToFile(filePath string, prefix string) error {
	return os.WriteFile(filePath, []byte(prefix+"\n\n"+pv.content), 0644)
}

// HasSpecialName returns false for PrintVisitor - no special naming conventions
func (pv *PrintVisitor) HasSpecialName(kind string) bool {
	return false
}

// GetName is not implemented for PrintVisitor
func (pv *PrintVisitor) GetName(tsNode *tree_sitter.Node) string {
	// PrintVisitor is for debugging, GetName not needed
	return ""
}
