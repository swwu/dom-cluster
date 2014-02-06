package cluster

import (
  "math/rand"
  "github.com/predictive-edge/dom-cluster/dom"
)

const MergeScoreCutoff = 0.3

type Template struct {
  Wrapper *dom.Node
  NumPages int
  Included []*dom.Node
  BaseUri string
  Uris []string
}

func NewTemplate(baseTemplate *dom.Node) *Template {
  return &Template{
    Wrapper: baseTemplate,
    NumPages: 1,
  }
}

func (t *Template) AddEntry(newEntry *dom.Entry) bool {
  newWrapper, score := NodeMerge(t.Wrapper, newEntry.Dom)
  if score < MergeScoreCutoff {
    t.NumPages++
    t.Wrapper = newWrapper
    t.Uris = append(t.Uris, newEntry.Uri)
    return true
  } else {
    return false
  }

  return score > 0
}

// returns true with probability p
func BernoulliChance(p float64) bool {
  return rand.Float64() < p
}

// pick a random key from the given map
func randomWrapper(wrappers map[*dom.Node]*dom.Entry) *dom.Node {
  reservoirCount := 0
  var pickedWrapper *dom.Node
  for wrapper := range wrappers {
    reservoirCount++
    if BernoulliChance( 1 / float64(reservoirCount) ) {
      pickedWrapper = wrapper
    }
  }
  return pickedWrapper
}

func DoCluster(entries []*dom.Entry) []*Template {
  templates := []*Template{}

  unusedWrappers := map[*dom.Node]*dom.Entry{}
  usedWrappers := map[*dom.Node]struct{}{}

  for _,entry:= range entries {
    unusedWrappers[entry.Dom] = entry
  }

  for len(unusedWrappers) > 0 {
    templateSeed := randomWrapper(unusedWrappers)
    seedEntry := unusedWrappers[templateSeed]
    usedWrappers[templateSeed] = struct{}{}
    delete(unusedWrappers, templateSeed)

    curTemplate := NewTemplate(templateSeed)
    curTemplate.BaseUri = seedEntry.Uri

    foundMore := true
    // since distance becomes shorter as the template becomes more general, we
    // need to recheck all elements every time the template is augmented
    for foundMore == true {
      foundMore = false
      // continue to iterate and add to the template until there are no more
      // things to add
      for wrapper := range unusedWrappers {
        if curTemplate.AddEntry(unusedWrappers[wrapper]) {
          usedWrappers[wrapper] = struct{}{}
          delete(unusedWrappers, wrapper)
          foundMore = true
        }
      }
    }

    templates = append(templates, curTemplate)
  }

  return templates

}
