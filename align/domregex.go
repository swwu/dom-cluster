package align

import (
  "github.com/predictive-edge/dom-cluster/dom"
)

type RegexNodeAlignment struct {
  score int
  aNodes []*dom.Node
  bNodes []*dom.Node
  aligned []AlignmentInstance
}

func NodeArrRegexAlign(a []*dom.Node, b []*dom.Node) *RegexNodeAlignment {
  return nil
}
