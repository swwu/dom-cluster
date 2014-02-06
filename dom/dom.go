package dom

import (
  "bytes"
  "strings"
  "fmt"
)

// non-numeric values for Node.Sign
const (
  NoSign = iota // equivalent to 1, since Sign defaults to 0 when unmarshaling
  OnePlus = -iota
  ZeroPlus = -iota
  ZeroOne = -iota
  Fixed = -iota
)


type Entry struct {
  Uri string `json:"url"`
  Dom *Node `json:"dom"`
}

type Node struct {

  NodeName string `json:"nodeName"`
  TagName string `json:"tagName"`

  Attrs map[string]string `json:"attrs"`

  Text string `json:"text"`

  Children []*Node `json:"children"`

  Sign int `json:"sign"`

  treeDepth int
  treeWeight int
  isParen bool

}

func DefaultNode() *Node {
  return &Node{
    Sign:1,
  }
}

func NewParenNode(children []*Node, sign int) *Node {
  return &Node {
    NodeName: "##paren",
    Children: children,
    Sign: sign,
    isParen: true,
  }
}

func (n *Node) TreeDepth() int {
  if n.treeDepth > 0 { return n.treeDepth }

  myDepth := 1 // the depth that this node counts for (0 for paren, 1 otherwise)
  if len(n.Children) > 0 {
    maxChildrenDepth := 0
    for _,child := range n.Children {
      if child.TreeDepth() > maxChildrenDepth {
        maxChildrenDepth = child.TreeDepth()
      }
    }
    n.treeDepth = myDepth + maxChildrenDepth
  } else {
    n.treeDepth = myDepth
  }
  return n.treeDepth
}

func (n *Node) TreeWeight() int {
  if n.treeWeight > 0 { return n.treeWeight }

  // if the sign allows this node to not exist, then its weight/alignment
  // cost is always zero
  if n.Sign == ZeroPlus || n.Sign == OnePlus {
    n.treeWeight = 0
    return 0
  }

  myWeight := 1 // the weight that this node counts for

  // if this is a paren node then it itself doesn't count for any weight
  if n.isParen {
    myWeight = 0
  }

  // weight = this node's weight + all child weights
  for _,c := range n.Children {
    myWeight += c.TreeWeight()
  }

  // constant repeated elements have the equivalent constant-factor multiples
  // on weight
  if n.Sign > 1 {
    myWeight *= n.Sign
  }

  return myWeight
}

func (n *Node) CallPreOrder(fn func(*Node)) {
  fn(n)
  for _,child := range n.Children {
    child.CallPreOrder(fn)
  }
}

func (n *Node) SignStr() string {
  switch {
  case n.Sign >= 0:
    return fmt.Sprintf("%d",n.Sign)
  case n.Sign == OnePlus:
    return "+"
  case n.Sign == ZeroPlus:
    return "*"
  case n.Sign == ZeroOne:
    return "?"
  default:
    return "_"
  }
}

func (n *Node) String() string {
  buf := bytes.NewBufferString(n.NodeName)

  if id,exists := n.Attrs["id"]; exists {
    buf.WriteString(fmt.Sprintf("#%s",id))
  }
  if classes,exists := n.Attrs["class"]; exists {
    buf.WriteString(
      fmt.Sprintf(".%s",strings.Join(strings.Split(classes, " "),".")))
  }
  buf.WriteString(fmt.Sprintf("^{%s}",n.SignStr()))

  return buf.String()
}


