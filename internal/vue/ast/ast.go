package vue_ast

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

type Namespace uint8

const (
  NamespaceHTML Namespace = iota
  NamespaceSVG
  NamespaceMATH_ML
)


type NodeType uint16

const (
  NodeTypeROOT NodeType = iota
  NodeTypeELEMENT
  NodeTypeTEXT
  NodeTypeCOMMENT
  NodeTypeSIMPLE_EXPRESSION
  NodeTypeINTERPOLATION
  NodeTypeATTRIBUTE
  NodeTypeDIRECTIVE
)

type Node struct {
	Type NodeType
	Loc core.TextRange
	data nodeData
}

type nodeData interface {
	AsNode() *Node
}

func (n *Node) AsNode() *Node {
	return n
}


type ElementType uint8

const (
  ElementTypeELEMENT ElementType = iota
  ElementTypeCOMPONENT
  ElementTypeSLOT
  ElementTypeTEMPLATE
)

type RootNode struct {
	Node
  Children []*Node // TemplateChildNode[]
}

func NewRootNode() *RootNode {
	data := RootNode{Node: Node{Type: NodeTypeROOT}}
	data.Node.data = &data
	return &data
}

type ElementNode struct {
	Node
  Ns Namespace
  Tag string
  TagType ElementType
  Props []*Node // Array<AttributeNode | DirectiveNode>
  Children []*Node // TemplateChildNode[]
  IsSelfClosing bool
	// Only for <script>
	Ast *ast.SourceFile
	// Only for SFC root level elements
	InnerLoc core.TextRange
}

func NewElementNode(ns Namespace, tag string, loc core.TextRange) *ElementNode {
	data := ElementNode{Node: Node{Type: NodeTypeELEMENT, Loc: loc}, Ns: ns, Tag: tag}
	data.Node.data = &data
	return &data
}

type ScriptElementNode struct {
	ElementNode
}

type TextNode struct {
	Node
  Content string
}

func NewTextNode(content string, loc core.TextRange) *TextNode {
	data := TextNode{Node: Node{Type: NodeTypeTEXT, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

type SimpleExpressionNode struct {
	Node
  Content string
	// nil when expression is a simple identifier (static)
	Ast *ast.SourceFile
	// TODO
  // isHandlerKey?: boolean
}

func NewSimpleExpressionNode(content string, ast *ast.SourceFile, loc core.TextRange) *SimpleExpressionNode {
	data := SimpleExpressionNode{Node: Node{Type: NodeTypeSIMPLE_EXPRESSION, Loc: loc}, Content: content, Ast: ast}
	data.Node.data = &data
	return &data
}

type CommentNode struct {
	Node
  Content string
}

func NewCommentNode(content string, loc core.TextRange) *CommentNode {
	data := CommentNode{Node: Node{Type: NodeTypeCOMMENT, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

type AttributeNode struct {
	Node
  Name string
  NameLoc core.TextRange
  Value *TextNode // | undefined
}

func NewAttributeNode(name string, nameLoc, loc core.TextRange) *AttributeNode {
	data := AttributeNode{Node: Node{Type: NodeTypeATTRIBUTE, Loc: loc}, Name: name, NameLoc: nameLoc}
	data.Node.data = &data
	return &data
}

type DirectiveNode struct {
	Node
  /**
   * the normalized name without prefix or shorthands, e.g. "bind", "on"
   */
  Name string
  /**
   * the raw attribute name, preserving shorthand, and including arg & modifiers
   * this is only used during parse.
   */
	RawName string // :?
  // exp ExpressionNode | undefined
  // arg ExpressionNode | undefined
  // modifiers: SimpleExpressionNode[]
}

func NewDirectiveNode(name, rawName string, loc core.TextRange) *DirectiveNode {
	data := DirectiveNode{Node: Node{Type: NodeTypeDIRECTIVE, Loc: loc}, Name: name, RawName: rawName}
	data.Node.data = &data
	return &data
}


type InterpolationNode struct {
	Node
	Content *SimpleExpressionNode
}

func NewInterpolationNode(content *SimpleExpressionNode, loc core.TextRange) *InterpolationNode {
	data := InterpolationNode{Node: Node{Type: NodeTypeINTERPOLATION, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

func (n *Node) AsElement() *ElementNode {
	return n.data.(*ElementNode)
}
func (n *Node) AsText() *TextNode {
	return n.data.(*TextNode)
}
func (n *Node) AsComment() *CommentNode {
	return n.data.(*CommentNode)
}
func (n *Node) AsSimpleExpression() *SimpleExpressionNode {
	return n.data.(*SimpleExpressionNode)
}
func (n *Node) AsInterpolation() *InterpolationNode {
	return n.data.(*InterpolationNode)
}
func (n *Node) AsAttribute() *AttributeNode {
	return n.data.(*AttributeNode)
}
func (n *Node) AsDirective() *DirectiveNode {
	return n.data.(*DirectiveNode)
}
