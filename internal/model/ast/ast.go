package ast

import (
	"github.com/armchr/codeapi/pkg/lsp/base"
)

type NodeType int8

const (
	NodeTypeModuleScope  NodeType = 1
	NodeTypeFileScope    NodeType = 2
	NodeTypeBlock        NodeType = 3
	NodeTypeVariable     NodeType = 4
	NodeTypeExpression   NodeType = 5
	NodeTypeConditional  NodeType = 6
	NodeTypeFunction     NodeType = 7
	NodeTypeClass        NodeType = 8
	NodeTypeField        NodeType = 9
	NodeTypeFunctionCall NodeType = 10
	NodeTypeFileNumber   NodeType = 11
	NodeTypeLoop         NodeType = 12
	NodeTypeImport       NodeType = 13
)

type NodeID int64

const (
	InvalidNodeID NodeID = 0
)

type Node struct {
	ID       NodeID         `json:"id"`
	NodeType NodeType       `json:"node_type"`
	FileID   int32          `json:"file_id"`
	Name     string         `json:"name,omitempty"`
	Range    base.Range     `json:"range"`
	Version  int32          `json:"version,omitempty"`
	ScopeID  NodeID         `json:"scope_id,omitempty"`
	MetaData map[string]any `json:"metadata,omitempty"`
}

func NewNode(
	id NodeID, nodeType NodeType, fileID int32,
	name string, rng base.Range, version int32, scopeID NodeID,
) *Node {
	return &Node{
		ID:       id,
		NodeType: nodeType,
		FileID:   fileID,
		Name:     name,
		Range:    rng,
		Version:  version,
		ScopeID:  scopeID,
	}
}
