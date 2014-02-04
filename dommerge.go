package main

import (
  "strings"
  "bytes"
  "io"
  "log"
  "fmt"
  "bufio"
  "encoding/json"
  "os"
  //"github.com/davecheney/profile"
)

const EditDistThreshold = 0.3

// non-numeric values for Node.Sign
const (
  NoSign = iota // equivalent to 1, since Sign defaults to 0 when unmarshaling
  OnePlus = -iota
  ZeroPlus = -iota
  ZeroOne = -iota
  Fixed = -iota
)

// possible node types
const (
  ContentNode = iota // normal dom node, has tagName etc
  ParenNode = iota // paren meta-node to apply signs to sibling node groups
  MetaNode = iota // node without structural influence (comments, scripts, etc)
)

const MaxUint = ^uint(0)
const MinUint = 0
const MaxInt = int(MaxUint >> 1)
const MinInt = -(MaxInt - 1)

// some utility functions

func Min(v ...int) int {
  ret := MaxInt
  for _,val := range v {
    if val < ret { ret = val }
  }
  return ret
}

// Filter returns a new slice holding only the elements of s that satisfy f()
func FilterRegionGroups(s []*regionGroup, fn func(*regionGroup) bool) []*regionGroup {
  var p []*regionGroup
  for _, v := range s {
    if fn(v) {
      p = append(p, v)
    }
  }
  return p
}


type AlignmentInstance struct {
  a *Node
  b *Node
}

type NodeAlignment struct {
  score int
  aligned []AlignmentInstance
}

// creates a "base" alignment (empty aligned array) with a given score
func BaseNodeAlignment(score int) *NodeAlignment {
  return &NodeAlignment {
    score: score,
  }
}

func (na *NodeAlignment) MakeCopy() *NodeAlignment {
  tmp := make([]AlignmentInstance, len(na.aligned))
  copy(tmp, na.aligned)
  return &NodeAlignment {
    score: na.score,
    aligned: tmp,
  }
}

// insertion
func (na *NodeAlignment) InsOp(score int, newNode *Node) {
  na.score += score
  na.aligned = append(na.aligned, AlignmentInstance{
    a: newNode,
    b: nil,
  })
}

// substitution
func (na *NodeAlignment) SubOp(score int, firstNode *Node, secondNode *Node) {
  na.score += score
  na.aligned = append(na.aligned, AlignmentInstance{
    a: secondNode,
    b: firstNode,
  })
}

// deletion
func (na *NodeAlignment) DelOp(score int, delNode *Node) {
  na.score += score
  na.aligned = append(na.aligned, AlignmentInstance{
    a: nil,
    b: delNode,
  })
}

func (na *NodeAlignment) PrintAlignment() {
  for _,alignment := range na.aligned {
    var aName = "---"
    var bName = "---"
    if alignment.a != nil {
      aName = alignment.a.String()
    }
    if alignment.b != nil {
      bName = alignment.b.String()
    }
    fmt.Printf("%s <----> %s\n", aName,bName)
  }
}


func NodeMerge(a,b *Node) *Node {
  newNode := DefaultNode()
  var newSign int

  // if they are both non-nil, then "merge" the nodes
  if a != nil && b != nil {
    if a.NodeName == b.NodeName {
      newNode.NodeName = a.NodeName
    } else {
      newNode.NodeName = "##mismatch"
    }
    alignment := NodeArrAlign(a.Children, b.Children)
    for _,instance := range alignment.aligned {
      newNode.Children = append(newNode.Children, NodeMerge(instance.a,instance.b))
    }

    aSign := a.Sign
    bSign := b.Sign

    if aSign == 0 { aSign = 1 }
    if bSign == 0 { bSign = 1 }
    if aSign < bSign { // a is always larger (special char constants are all negative)
      tmp := aSign
      aSign = bSign
      bSign = tmp
    }

    // handle signs
    switch {
    // 1,1 -> 1
    // N,N -> N
    // *,* -> *
    // +,+ -> +
    // ?,? -> ?
    case aSign == bSign:
      newSign = aSign

    // 1,? -> ?
    case aSign == 1 && bSign == ZeroOne:
      newSign = ZeroOne

    // N,? -> *
    case aSign > 1 && bSign == ZeroOne,
    // +,? -> *
      aSign == OnePlus && bSign == ZeroOne,
    // 1,* -> *
    // N,* -> *
      aSign >= 1 && bSign == ZeroPlus,
    // *,? -> *
      aSign == ZeroPlus && bSign == ZeroOne,
    // +,* -> *
      aSign == OnePlus && bSign == ZeroPlus:
      newSign = ZeroPlus

    // N,1|N,M -> +
    case aSign > 1 && bSign >=1,
    // N,+|1,+ -> +
      aSign >= 1 && bSign == OnePlus:
      newSign = OnePlus
    default:
      log.Fatal("ordering wrong somewhere")
    }

  // if only one is non-nil, then add that node with transformed sign
  } else if a != nil || b != nil {
    if a != nil {
      newNode.NodeName = a.NodeName
    } else if b != nil {
      newNode.NodeName = b.NodeName
    }

    var sign int
    var node *Node
    if a == nil {
      node = b
      sign = b.Sign
    } else {
      node = a
      sign = a.Sign
    }
    if sign == 0 { sign = 1 }

    newNode.Children = node.Children

    // handle signs
    switch {
    // 1|? -> ?
    case sign == 1 || sign == ZeroOne:
      newSign = ZeroOne
    //+|*|N -> *
    default:
      newSign = ZeroPlus
    }
  }

  newNode.Sign = newSign

  return newNode
}

// align two forests
// TODO: we can't ultimately use the matrix approach because ?+* prevents us
// from knowing beforehand the dimensions of the matrix
func NodeArrAlign(a,b []*Node) *NodeAlignment {
  la := len(a)
  lb := len(b)

  d  := make([]*NodeAlignment, la + 1) // single array (current column of levenshtein matrix)

  // initialize first column
  d[0] = BaseNodeAlignment(0)
  for i := 1; i <= la; i++ {
    d[i] = d[i-1].MakeCopy()
    d[i].InsOp(a[i-1].TreeWeight(), a[i-1])
  }

  // iteration of i corresponds to "moving" the column represented by d,
  // where at the end of each loop d represents the ith column of the matrix.
  // before the first iteration d represents the 0th column
  for i := 1; i <= lb; i++ {
    // the 0th element of each column is just the total "cost" of b so far
    lastDiag := d[0] // keep track of the diagonal to substitute from
    d[0] = d[0].MakeCopy()
    d[0].InsOp(b[i-1].TreeWeight(), b[i-1])

    // we now iterate over the column
    for j := 1; j <= la; j++ {
      lastVal := d[j] // this is the value of this row in the previous column

      // we now calculate horizontal, vertical, and diagonal movement in the
      // matrix. horizontal movement represents an addition, vertical
      // represents a deletion, and diagonal represents a substitution

      // TODO: properly adjust signs, make sure "missing" elements are
      // properly inserted (a "deletion" still inserts the deleted element
      // with sign '?')

      // horizontal movement (deletion)
      newVal := d[j].MakeCopy()
      newVal.DelOp(b[i-1].TreeWeight(), b[i-1])

      // vertical movement (insertion)
      vVal := d[j-1].MakeCopy()
      vVal.InsOp(a[j-1].TreeWeight(), a[j-1])
      if vVal.score < newVal.score {
        newVal = vVal
      }

      // diagonal movement
      subCost := 0
      if a[j-1].NodeName != b[i-1].NodeName {
        // make it more expensive than ins + del 
        subCost = a[j-1].TreeWeight() + b[i-1].TreeWeight() + 1
      }
      dVal := lastDiag.MakeCopy()
      dVal.SubOp(subCost, a[j-1], b[i-1])
      if dVal.score < newVal.score {
        newVal = dVal
      }

      d[j] = newVal

      lastDiag = lastVal // update the diagonal to be the element formerly to the left
    }
  }

  return d[la]
}

// Levenshtein to compare tag arrays
func TagLevenshteinDistance(a, b []string) int {
  la := len(a)
  lb := len(b)
  d  := make([]int, la + 1)
  var lastdiag, olddiag, temp int

  for i := 1; i <= la; i++ {
    d[i] = i
  }
  for i := 1; i <= lb; i++ {
    d[0] = i
    lastdiag = i - 1
    for j := 1; j <= la; j++ {
      olddiag = d[j]
      min := d[j] + 1
      if (d[j - 1] + 1) < min {
        min = d[j - 1] + 1
      }
      if ( a[j - 1] == b[i - 1] ) {
        temp = 0
      } else {
        temp = 1
      }
      if (lastdiag + temp) < min {
        min = lastdiag + temp
      }
      d[j] = min
      lastdiag = olddiag
    }
  }
  return d[la]
}




type Entry struct {
  Url string `json:"url"`
  Dom Node `json:"dom"`
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

type repeatGroup struct {
  RegionGroups []*regionGroup
}

type regionGroup struct {
  Regions []*nodeRegion
}

type nodeRegion struct {
  Nodes []*Node
}

func (nr *nodeRegion) String() string {
  buf := bytes.NewBufferString("region - ")

  for _,n := range nr.Nodes {
    buf.WriteString(fmt.Sprintf("%s.%v^{%d}",n.NodeName,n.Attrs["class"],n.Sign))
  }

  return buf.String()
}

/*
The node templatization algorithm.

Creates a compact, text-agnostic representation of the given DOM tree.
Specifically, detects nodes and node groups (adjacent sibling nodes) with
similar or identical structure, and represents them as a single signed node
or signed parenthetical node whenever possible
*/
func TemplatizeNode(node *Node, k int) *Node {
  // since we operate on the node's children to find repeating sibling groups,
  // depths of 1 or 2 are extremely unlike
  if node.TreeDepth() >= 3 {
    // if it's not a paren node, try to find repeating elements
    if node.NodeName != "##paren" {
      // find any repeating patterns in this node's children
      // a "group" represents the repeating element of such a pattern

      allGroups := combComp(node.Children, k)

      //prelen := len(node.Children)

      // every node that's alraedy in a group
      groupedChildren := map[*Node]struct{}{}
      for _,group := range allGroups {
        for _,region := range group.Regions {
          for _,curNode := range region.Nodes {
            groupedChildren[curNode] = struct{}{}
          }
        }
      }

      newChildren := []*Node{}
      for _,c := range node.Children {
        // if it hasn't been grouped, just add it in its original place
        if _,exists := groupedChildren[c]; !exists {
          newChildren = append(newChildren, c)
        } else {
          // otherwise, check if a group starts with it. if it doesn't, then add
          // that entire group here
          for _,group := range allGroups {
            if c == group.Regions[0].Nodes[0] {
              if len(group.Regions[0].Nodes) == 1 {
                // if it's a grouping of 1-sized regions then just add the
                // first element with sign = len(group.Regions)
                // TODO: generate a copy of the node in question / create it
                // by properly merging all the group's regions together
                c.Sign = len(group.Regions)
                newChildren = append(newChildren, c)
              } else {
                // if it's a grouping of non-1-sized regions then just add the
                // first region as a paren group with sign = group.length
                newChildren = append(newChildren,
                NewParenNode(group.Regions[0].Nodes,len(group.Regions)))
              }
            }
          }
        }
      }
      //oldChildren := node.Children
      node.Children = newChildren
      //postlen := len(node.Children)

      //fmt.Println("==========")
      //fmt.Println(listToTagArr(node.Children,10))
      //if prelen != postlen {
      //  fmt.Printf("%d to %d\n", prelen, postlen)
      //}

    }
    for _,c := range node.Children {
      TemplatizeNode(c, k)
    }
  }

  return node
}

func listToTagArr (nodeList []*Node, k int) []string {
  retArr := []string{}
  for _,node := range nodeList {
    node.CallPreOrder(func (curNode *Node) {
      retArr = append(retArr, curNode.NodeName)
    })
  }
  return retArr
}

func tagArrSimilar(tagArr1 []string, tagArr2 []string) bool {
  // if the lengths differ by enough to make the threshold impossible to reach,
  // just return false
  // TODO: this is technically a less strict requirement than our actual cutoff
  if len(tagArr1) == len(tagArr2) && len(tagArr1) == 0 {
    return true
  }
  if float32(len(tagArr1)) / float32(len(tagArr2)) <= 1-EditDistThreshold ||
  float32(len(tagArr2)) / float32(len(tagArr1)) <= 1-EditDistThreshold {
    return false
  }

  return float32(TagLevenshteinDistance(tagArr1,tagArr2))/(float32(len(tagArr1)+len(tagArr2))*0.5) <=
  EditDistThreshold
}

func combComp(nodeList []*Node, k int) []*regionGroup {
  // no groups possible if there's only one node
  if len(nodeList) <= 1 {
    return []*regionGroup{}
  }

  k = Min(k,int(len(nodeList)/2))
  allGroups := []*repeatGroup{}

  for regionSize:=1; regionSize<=k; regionSize++ {
    curRepeatGroup := &repeatGroup{}
    allGroups = append(allGroups, curRepeatGroup)
    for offset:=0;offset<regionSize;offset++ {
      regionStart := offset

      // initialize the current group that we're building. more than one group
      // can be built if they are not contiguous
      curGroup := &regionGroup{}

      // initialize array values for iteration
      nextRegion := &nodeRegion{
        Nodes: nodeList[regionStart:regionStart+regionSize],
      }
      nextTags := listToTagArr(nextRegion.Nodes, k)

      // continue to iterate as long as the next region exists in whole
      for regionStart + 2*regionSize <= len(nodeList) {
        thisRegion := nextRegion
        nextRegion = &nodeRegion{
          Nodes: nodeList[regionStart+regionSize:regionStart+2*regionSize],
        }
        thisTags := nextTags
        nextTags = listToTagArr(nextRegion.Nodes, k)

        if tagArrSimilar(thisTags, nextTags) {
          if len(curGroup.Regions) > 0 && curGroup.Regions[len(curGroup.Regions)-1] != thisRegion {
            curRepeatGroup.RegionGroups = append(curRepeatGroup.RegionGroups,curGroup)
            curGroup = &regionGroup{}
          }
          if len(curGroup.Regions) == 0 {
            curGroup.Regions = append(curGroup.Regions, thisRegion)
          }
          curGroup.Regions = append(curGroup.Regions, nextRegion)
        }
        regionStart += regionSize
      }
      if len(curGroup.Regions) > 0 {
        curRepeatGroup.RegionGroups = append(curRepeatGroup.RegionGroups,curGroup)
      }
    }
  }

  retGroups := []*regionGroup{}
  hasGroupIntersect := func (g1, g2 *regionGroup) bool {
    hash := map[*Node]struct{}{}
    for _,gg1 := range g1.Regions {
      for _,node := range gg1.Nodes {
        hash[node] = struct{}{}
      }
    }
    for _,gg2 := range g2.Regions {
      for _,node := range gg2.Nodes {
        if _,exists := hash[node]; exists {
          return true
        }
      }
    }
    return false
  }

  // the order of iteration is important
  for groupSize:=k;groupSize>=1;groupSize-- {
    sizedGroups := allGroups[groupSize-1] // allGroups[0] has size 1, etc

    // remove all groups that intersect with this one, then add this one.
    //this allows smaller groups to take precedence over larger ones, so that e.g.
    // AAAA will properly be identified as A^4 rather than (AA)^2
    for _,group := range sizedGroups.RegionGroups {
      // TODO: make longer regionGroups of each given regionSize have precedence

      retGroups = FilterRegionGroups(retGroups, func(retGroup *regionGroup) bool {
        return !hasGroupIntersect(retGroup,group)
      })
      retGroups = append(retGroups, group)
    }
  }

  return retGroups
}


func getEntries(filename string) []*Entry {
  f, err := os.Open(filename)
  if err != nil {
    log.Fatal(err)
  }
  defer f.Close()

  r := bufio.NewReaderSize(f,512*1024)
  r.ReadLine() // extra read since first one is template
  line, isPrefix, err := r.ReadLine()
  entries := []*Entry{}
  for err == nil && !isPrefix {
    var entry Entry
    json.Unmarshal(line, &entry)

    entries = append(entries, &entry)
    line, isPrefix, err = r.ReadLine()
  }
  if err != io.EOF && err != nil {
      log.Fatal(err)
  }
  if isPrefix {
    log.Fatal("buffer size too small")
  }

  return entries
}

func main() {
  //defer profile.Start(profile.CPUProfile).Stop()

  entries := getEntries("./test.txt")
  //entry := entries[0]
  //templatizedNode := TemplatizeNode(&entry.Dom,10)
  //fmt.Println(templatizedNode)
  a := TemplatizeNode(&entries[2].Dom,10)
  b := TemplatizeNode(&entries[1].Dom,10)
  fmt.Println(NodeMerge(a,b).Children)
}
