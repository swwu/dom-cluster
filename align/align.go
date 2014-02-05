package align

import (
  "fmt"
  "github.com/predictive-edge/dom-cluster/dom"
  //"github.com/davecheney/profile"
)

type AlignmentInstance struct {
  a *dom.Node
  b *dom.Node
}

func (i *AlignmentInstance) A() *dom.Node {
  return i.a
}

func (i *AlignmentInstance) B() *dom.Node {
  return i.b
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

func (na *NodeAlignment) Score() int {
  return na.score
}

func (na *NodeAlignment) Aligned() []AlignmentInstance {
  return na.aligned
}

// insertion
func (na *NodeAlignment) InsOp(score int, newNode *dom.Node) {
  na.score += score
  na.aligned = append(na.aligned, AlignmentInstance{
    a: newNode,
    b: nil,
  })
}

// substitution
func (na *NodeAlignment) SubOp(score int, firstNode *dom.Node, secondNode *dom.Node) {
  na.score += score
  na.aligned = append(na.aligned, AlignmentInstance{
    a: secondNode,
    b: firstNode,
  })
}

// deletion
func (na *NodeAlignment) DelOp(score int, delNode *dom.Node) {
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



// align two forests
// TODO: we can't ultimately use the matrix approach because ?+* prevents us
// from knowing beforehand the dimensions of the matrix
func NodeArrAlign(a,b []*dom.Node) *NodeAlignment {
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
