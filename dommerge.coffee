extend = require 'extend'

fs = require 'fs'

global = window? or global

# weight for structural node edit-distance
DIFF_NODE_DIST = 1000
# weight for text-node content edit-distance
DIFF_TEXT_DIST = 1
# damping coefficient per level of the DOM tree
LEVEL_DIST_MULTIPLIER = 1/1.1
# required (but non-semantic) formatting elements like tbody get a pass
LEVEL_DIST_FORMATONLY_MULTIPLIER = 1


# token used to concatenate memoize_tokens for memoization of n-arg fns
MEMOIZE_CONCAT_TOKEN = "|||"


uuid = ->
  "#{(new Date()).getTime()}#{Math.random().toString(36).substring(2)}"

zip = (args...) ->
  lengthArray = (arr.length for arr in args)
  length = Math.min(lengthArray...)
  for i in [0...length]
    arr[i] for arr in args

sum = (arr) ->
  return arr.reduce(((x,y) -> x+y), 0)

copy = (obj) ->
  return extend(Object.create(null), obj)

###
memoizeNodes = (fn) ->
  memoObj = Object.create(null)
  (nodes...) ->
    memoToken = (for node in nodes
      if not node.memoize_token
        node.memoize_token = Math.random().toString(36).substring(2)
    ).sort().join('|||')
    if memoToken of memoObj
      return memoObj[memoToken]
    else
      retVal = fn.apply(global,nodes)
      memoObj[memoToken] = retVal
      return retVal
###


class Node
  is_filler: false
  memoize_token: false
  nodeName: null
  tagName: null
  attrs: null
  text: null
  children: null

  editWeight: null

  constructor: (nodeDict) ->

    # filler nodes are those that differ between two merged nodes
    @is_filler = false

    # memoize_token is used to memoize node operations
    @memoize_token = uuid()

    if not nodeDict then nodeDict = Object.create(null)

    # things that describe the node itself
    @nodeName = nodeDict.nodeName
    @tagName = nodeDict.tagName
    @attrs = nodeDict.attrs
    @text = nodeDict.text
    if nodeDict.children
      @children = for node in nodeDict.children
        new Node(node)
    else
      @children = []

  getText: ->
    if @nodeName == "#text" and !@is_filler
      return @text
    else
      return null

  getLevelMultiplier: ->
    if @nodeName in ["tbody"]
      return LEVEL_DIST_FORMATONLY_MULTIPLIER
    else
      return LEVEL_DIST_MULTIPLIER

  getEditWeight: ->
    if !@editWeight
      @editWeight = 0
      if @children
        @editWeight = @getLevelMultiplier() * sum(c.getEditWeight() for c in @children)
      @editWeight += if @nodeName == "#text"
        DIFF_TEXT_DIST
      else
        DIFF_NODE_DIST

    return @editWeight

  # returns a shallow copy
  makeCopy: ->
    retObj = extend(Object.create(null), @)
    retObj.children = []
    retObj.editWeight = null
    return retObj

  toDict: ->
    return {
      "nodeName": @nodeName,
      "tagName": @tagName,
      "attrs": @attrs,
      "text": @text,
      "children": c.toDict() for c in (@children or [])
    }

# TextFillerNode indicates that text has changed but structure has not
class TextFillerNode extends Node
  constructor: ->
    @is_filler = true
    @memoize_token = uuid()

    @nodeName = "#text"
    @tagName = null

# DomFillerNode indicates that structure has changed
class DomFillerNode extends Node
  constructor: ->
    @is_filler = true
    @memoize_token = uuid()

    @nodeName = "#filler"
    @tagName = null


# edit distance between two nodes
nodeDist = (node1, node2) ->
  mergeNodes(node1, node2).dist / (node1.getEditWeight() + node2.getEditWeight())

nodeMerge = (node1, node2) -> mergeNodes(node1, node2).dom

mergeNodes = (node1, node2) ->
  # if nodeNames differ, return structure-filler
  if node1.nodeName != node2.nodeName
    return {
      dom: new DomFillerNode()
      dist: node1.getEditWeight() + node2.getEditWeight()
    }
  # guaranteed that node1.nodeName == node2.nodeName from this point on

  # if both fillers, just return a filler of the appropriate type
  if node1.is_filler and node2.is_filler
    if node1.nodeName == "#text"
      return {
        dom: new TextFillerNode()
        dist: 0
      }
    else
      return {
        dom: new DomFillerNode()
        dist: 0
      }

  # if both non-filler text nodes,
  # otherwise just return the text node
  if node1.nodeName == "#text"
    # generate a filler text node if text differs
    if node1.getText() != node2.getText()
      return {
        dom: new TextFillerNode()
        dist: 2*DIFF_TEXT_DIST
      }
    # return the (shallow-copied) same text node if text doesn't differ
    else
      return {
        dom: node1.makeCopy()
        dist: 0
      }

  for attr in ["class", "id"]
    if node1[attr] != node2[attr]
      return {
        dom: new DomFillerNode()
        dist: node1.getEditWeight() + node2.getEditWeight()
      }

  # otherwise we will keep this node but merge its children
  retNode = node1.makeCopy()
  retDist = 0

  # if either node has children, merge children and get scores
  n1c = node1.children or []
  n2c = node2.children or []
  if n1c.length == 0 or n2c.length == 0
    retNode.children = []
    retDist += sum(c.getEditWeight() for c in n1c) + sum(c.getEditWeight() for c in n2c)
  else
    mergedChildObjs = for c in zip(n1c, n2c)
      mergeNodes(c[0],c[1])

    retNode.children = for o in mergedChildObjs
      o.dom 
    retDist += sum(for o in mergedChildObjs
      o.dist)
    slicePoint = retNode.children?.length or 0
    retDist += (sum(c.getEditWeight() for c in n1c[slicePoint..]) +
      sum(c.getEditWeight() for c in n2c[slicePoint..]))

  retDist *= retNode.getLevelMultiplier()

  return {
    dom: retNode
    dist: retDist
  }


mergeNodeArrs = (nodes1, nodes2) ->
  mergeNodeArrsRecurse = (i,j) ->

textEq = (c1, c2) ->
  return if c1 == c2 then 0 else 1
domEq = (node1, node2) ->
  return if node1.nodeName == node2.nodeName and
      node1.attrs?.id == node2.attrs?.id and
      node1.attrs?.class == node2.attrs?.class
    0
  else
    1

editDistance = (arr1, arr2, eqFn) ->
  editDistanceRecurse = (i, j) ->
    if i > arr1.length and j > arr2.length
      return 0
    if i > arr1.length
      return arr2.length - j
    if j > arr2.length
      return arr1.length - i

    return Math.min(
      editDistanceRecurse(i,j+1),
      editDistanceRecurse(i+1,j),
      editDistanceRecurse(i+1,j+1)
    ) + eqFn(arr1[i], arr2[j])

  return editDistanceRecurse(0,0)

similarity = (arr1, arr2, eqFn) ->
  return 1 - editDistance(arr1,arr2,eqFn)/Math.max(arr1.length, arr2.length)

###
fs.readFile('prods.txt', 'utf8', (err, data) ->
  entries = for pageText in data.split('\n').filter((x) -> x.length > 0)
    JSON.parse(pageText).dom
  val = entries.reduce((lhs, rhs) ->
    nodeMerge(lhs, rhs)
  , entries[0])

  console.log "var dom = #{JSON.stringify val}"
)
###
module.exports = {
  Node: Node
  nodeMerge: nodeMerge
  nodeDist: nodeDist
  mergeNodes: mergeNodes
}


