package cluster

import (
  "github.com/predictive-edge/dom-cluster/dom"
)

const MergeScoreCutoff = 0.3

type Template struct {
  Wrapper *dom.Node
  NumPages int
}

func NewTemplate(baseTemplate *dom.Node) *Template {
  return &Template{
    Wrapper: baseTemplate,
    NumPages: 1,
  }
}

func (t *Template) AddWrapper(newWrapper *dom.Node) bool {
  var score float64
  newWrapper, score := NodeMerge(t.Wrapper, newWrapper)
  if score < MergeScoreCutoff {
    t.NumPages++
    t.Wrapper = newWrapper
    return true
  } else {
    return false
  }

  return score > 0
}

