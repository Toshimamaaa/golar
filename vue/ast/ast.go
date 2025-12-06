package ast

import (
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
  // innerLoc?: SourceLocation // only for SFC root level elements
}

func NewElementNode(ns Namespace, tag string, loc core.TextRange) *ElementNode {
	data := ElementNode{Node: Node{Type: NodeTypeELEMENT, Loc: loc}, Ns: ns, Tag: tag}
	data.Node.data = &data
	return &data
}

// func NewElementNode(ns Namespace)

type TextNode struct {
	Node
  Content string
}

func NewTextNode(content string, loc core.TextRange) *TextNode {
	data := TextNode{Node: Node{Type: NodeTypeTEXT, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

/**
 * Static types have several levels.
 * Higher levels implies lower levels. e.g. a node that can be stringified
 * can always be hoisted and skipped for patch.
 */
type ConstantType uint8

const (
  ConstantTypeNOT_CONSTANT ConstantType = iota
  ConstantTypeCAN_SKIP_PATCH
  ConstantTypeCAN_CACHE
  ConstantTypeCAN_STRINGIFY
)

type SimpleExpressionNode struct {
	Node
  Content string
  IsStatic bool
  ConstType ConstantType
  /**
   * - `null` means the expression is a simple identifier that doesn't need
   *    parsing
   * - `false` means there was a parsing error
   */
  // ast?: BabelNode | null | false
  /**
   * Indicates this is an identifier for a hoist vnode call and points to the
   * hoisted node.
   */
  // hoisted?: JSChildNode
  /**
   * an expression parsed as the params of a function will track
   * the identifiers declared inside the function body.
   */
  // identifiers?: string[]
  // isHandlerKey?: boolean
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
	Content string // TODO: *ExpressionNode
}

func NewInterpolationNode(content string, loc core.TextRange) *InterpolationNode {
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
