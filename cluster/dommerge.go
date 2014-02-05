package cluster

import (
  "bytes"
  "log"
  "fmt"
  "github.com/predictive-edge/dom-cluster/align"
  "github.com/predictive-edge/dom-cluster/dom"
  //"github.com/davecheney/profile"
)

const EditDistThreshold = 0.3

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


// merges two nodes by aligning their children, properly synthesizing their
// signs, and then recursing on each aligned child pair
func NodeMerge(a,b *dom.Node) (*dom.Node,float64) {
  retNode, score := NodeMergeRecurse(a,b)

  // norm = score / ((total1 + total2)/2)
  normScore := 2*float64(score) / float64(a.TreeWeight() + b.TreeWeight())

  return retNode, normScore
}


// the inner recursive function that does the work for NodeMerge
func NodeMergeRecurse(a,b *dom.Node) (*dom.Node,int) {
  newNode := dom.DefaultNode()
  alignScore := 0
  var newSign int

  // if they are both non-nil, then "merge" the nodes
  if a != nil && b != nil {
    if a.NodeName == b.NodeName { // equal nodenames, we align
      newNode.NodeName = a.NodeName

      alignment := align.NodeArrAlign(a.Children, b.Children)
      alignScore += alignment.Score()
      for _,instance := range alignment.Aligned() {
        merged, mergeScore := NodeMergeRecurse(instance.A(),instance.B())
        alignScore += mergeScore
        newNode.Children = append(newNode.Children, merged)
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
      case aSign == 1 && bSign == dom.ZeroOne:
        newSign = dom.ZeroOne

      // N,? -> *
      case aSign > 1 && bSign == dom.ZeroOne,
      // +,? -> *
        aSign == dom.OnePlus && bSign == dom.ZeroOne,
      // 1,* -> *
      // N,* -> *
        aSign >= 1 && bSign == dom.ZeroPlus,
      // *,? -> *
        aSign == dom.ZeroPlus && bSign == dom.ZeroOne,
      // +,* -> *
        aSign == dom.OnePlus && bSign == dom.ZeroPlus:
        newSign = dom.ZeroPlus

      // N,1|N,M -> +
      case aSign > 1 && bSign >=1,
      // N,+|1,+ -> +
        aSign >= 1 && bSign == dom.OnePlus:
        newSign = dom.OnePlus

      default:
        log.Fatal("ordering wrong somewhere")
      }
    } else { // if we have a mismatch then score it as such
      newNode.NodeName = "##mismatch"
      alignScore += a.TreeWeight() + b.TreeWeight()
    }

  // if only one is non-nil, then add that node with transformed sign
  } else if a != nil || b != nil {
    if a != nil {
      newNode.NodeName = a.NodeName
    } else if b != nil {
      newNode.NodeName = b.NodeName
    }

    var sign int
    var node *dom.Node
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
    case sign == 1 || sign == dom.ZeroOne:
      newSign = dom.ZeroOne
    //+|*|N -> *
    default:
      newSign = dom.ZeroPlus
    }
  }

  newNode.Sign = newSign

  return newNode,alignScore
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




type repeatGroup struct {
  RegionGroups []*regionGroup
}

type regionGroup struct {
  Regions []*nodeRegion
}

type nodeRegion struct {
  Nodes []*dom.Node
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
func TemplatizeNode(node *dom.Node, k int) *dom.Node {
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
      groupedChildren := map[*dom.Node]struct{}{}
      for _,group := range allGroups {
        for _,region := range group.Regions {
          for _,curNode := range region.Nodes {
            groupedChildren[curNode] = struct{}{}
          }
        }
      }

      newChildren := []*dom.Node{}
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
                dom.NewParenNode(group.Regions[0].Nodes,len(group.Regions)))
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

func listToTagArr (nodeList []*dom.Node, k int) []string {
  retArr := []string{}
  for _,node := range nodeList {
    node.CallPreOrder(func (curNode *dom.Node) {
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
  if float64(len(tagArr1)) / float64(len(tagArr2)) <= 1-EditDistThreshold ||
  float64(len(tagArr2)) / float64(len(tagArr1)) <= 1-EditDistThreshold {
    return false
  }

  return float64(TagLevenshteinDistance(tagArr1,tagArr2))/(float64(len(tagArr1)+len(tagArr2))*0.5) <=
  EditDistThreshold
}

func combComp(nodeList []*dom.Node, k int) []*regionGroup {
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
    hash := map[*dom.Node]struct{}{}
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



