extend = require 'extend'
fs = require 'fs'
levenshtein = require 'fast-levenshtein'

global = window? or global

uuid = ->
  "#{(new Date()).getTime()}#{Math.random().toString(36).substring(2)}"
# the maximum normalized edit distance allowed for nodeListEditDist to consider two nodes structurally identical
EDIT_DIST_THRESHOLD = 0.3

NodeTypes = {
  TAG: 0
  TEXT: 1
  PAREN: 2
}


class BaseNode
  ###
  # Sign can be any natural number, or one of ?+*
  #
  # <number> - this node occurs exactly <number> times
  # ? - this node occurs either zero or one time
  # + - this node occurs one or more times
  # * - this node occurs zero or more times
  #
  # numbers will be integers, other symbols will be strings
  ###
  sign: 1

  nodeName: null
  tagName: null
  attrs: null
  text: null
  children: null

  editWeight: null

  constructor: (nodeDict) ->
    # memoize_token is used to memoize node operations
    @uuid = uuid()

  # does this node have a constant sign, i.e. does it always appear a fixed
  # number of times? constant signs are integers only
  isConstantSign: ->
    return !isNaN(@sign)

  # how deep the subtree rooted at this node is
  getTreeDepth: ->
    if @children?.length > 0
      return Math.max.apply(null, (c.getTreeDepth() for c in @children)) + 1
    else
      return 1

  # calls the function with each node as an argument in a pre-order traversal
  callPreOrder: (fn) ->
    fn(@)
    if @children
      for c in @children
        c.callPreOrder(fn)

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
      "sign": @sign,
      "children": c.toDict() for c in (@children or [])
    }

  toString: ->
    val = @nodeName
    if @attrs?.id
      val += "##{@attrs.id}"
    if @attrs?.class
      val += ".#{@attrs.class.split(" ").join(".")}"
    if @children
      val += "[#{@children.length}]"
    val += "^{#{@sign}}"
    return val

class Node extends BaseNode
  constructor: (nodeDict) ->
    super()

    if not nodeDict then nodeDict = Object.create(null)

    # things that describe the node itself
    @nodeName = nodeDict.nodeName
    @tagName = nodeDict.tagName
    @attrs = nodeDict.attrs
    @text = nodeDict.text
    @sign = 1

    if nodeDict.children
      @children = for node in nodeDict.children
        new Node(node)
    else
      @children = []


class ParenNode extends BaseNode
  constructor: (children, sign) ->
    super()

    @nodeName = "##paren"
    @sign = sign

    @tagName = null
    @attrs = null
    @text = null

    @children = children

  getTreeDepth: ->
    if @children?.length > 0
      return Math.max.apply(null, (c.getTreeDepth() for c in @children))
    else
      return 0


###
# The node templatization algorithm.
#
# Creates a compact, text-agnostic representation of the given DOM tree.
# Specifically, detects nodes and node groups (adjacent sibling nodes) with
# similar or identical structure, and represents them as a single signed node
# or signed parenthetical node whenever possible
###
templatizeNode = (node, k = 10) ->
  # since we operate on this node's children, depths of 1 or 2 are extremely
  # unlikely to find results
  if node.getTreeDepth() >= 3

    # if it's not a paren node, then try to find repeating elements
    if node.nodeName != "##paren"
      # find any repeating patterns in this node's children
      # allgroups is [ [ [node,node...],[node,node...] ], ... ]
      # each group is guaranteed to be contiguous, and groups are guaranteed not
      # to overlap
      allGroups = combComp(node.children, k)

      prelen = node.children.length
      preStr = node.children.toString()

      # find every node that's been added to a group
      groupedChildren = Object.create(null)
      for group in allGroups
        for region in group
          for curNode in region
            groupedChildren[curNode.uuid] = curNode

      newChildren = []
      for c in node.children
        # if it hasn't been grouped, just add it in its usual place
        if c.uuid not of groupedChildren
          newChildren.push(c)
        else
          # otherwise, check if a group starts with it. if it does, then add
          # that entire group here
          for group in allGroups
            if c == group[0][0]
              if group[0].length == 1
                # if it's a grouping of 1-sized regions then just add the first
                # element with sign = group.length
                c.sign = group.length
                newChildren.push(c)
              else
                # if it's a grouping of non-1-sized regions then just add the
                # first region as a paren group with sign = group.length
                newChildren.push(new ParenNode(group[0],group.length))

      oldChildren = node.children
      node.children = newChildren
      postlen = node.children.length

      if prelen != postlen
        console.log "#{prelen} to #{postlen}"
        console.log preStr
        console.log node.children.toString()
    # then recurse
    for c in node.children
      templatizeNode(c, k)

  return node


combComp = (nodeList, k = 10) ->
  listToTagArr = (nodeList) ->
    retArr = []
    for node in nodeList
      node.callPreOrder((curNode) ->
        retArr.push(curNode.nodeName)
      )
    return retArr

  if nodeList.length <= 1 then return []

  k = Math.min(k, Math.floor(nodeList.length/2))
  allGroups = []
  for regionSize in [1..k]
    allGroups[regionSize] = []
    for offset in [0...regionSize] # try all offsets up to k
      regionStart = offset
      #console.log "##size #{regionSize} offset #{offset} listlen #{nodeList.length}"

      # initialize the current group that we're building. more than one group
      # can be built if they are not contiguous
      curGroup = []

      # initialize array values for iteration
      nextRegion = nodeList[regionStart...regionStart+regionSize]
      nextTags = listToTagArr(nextRegion)

      # continue to iterate as long as the next region completely exists
      while regionStart+2*regionSize <= nodeList.length
        #console.log "#regionstart #{regionStart}"
        thisRegion = nextRegion
        nextRegion = nodeList[regionStart+regionSize...regionStart+2*regionSize]
        thisTags = nextTags
        nextTags = listToTagArr(nextRegion)
        #console.log tagArrSimilar(thisTags, nextTags)
        if tagArrSimilar(thisTags, nextTags)
          if curGroup[curGroup.length-1] != thisRegion and curGroup.length > 0
            allGroups[regionSize].push(curGroup)
            curGroup = []
          if curGroup.length == 0
            curGroup.push(thisRegion)
          curGroup.push(nextRegion)
        regionStart += regionSize

      if curGroup.length > 0
        allGroups[regionSize].push(curGroup)

  retGroups = []
  hasGroupIntersect = (g1, g2) ->
    hash = Object.create(null)
    for gg1 in g1
      for node in gg1
        hash[node.uuid] = node
    for gg2 in g2
      for node in gg2
        if node.uuid of hash
          return true
    return false
  # do it this way because it's important we check groups in order of
  # decreasing size
  for groupSize in [k..1]
    sizedGroups = allGroups[groupSize]
    if not sizedGroups then continue
    # remove all groups that intersect with this one, then add this one this
    # allows smaller groups to take precedence over larger ones, so that e.g.
    # AAAA will properly be identified as A^4 rather than (AA)^2
    for group in sizedGroups
      retGroups = retGroups.filter((retGroup) ->
        !hasGroupIntersect(retGroup, group)
      )
      retGroups.push(group)

  return retGroups


tagArrSimilar = (tagArr1, tagArr2) ->
  # if the lengths differ by enough to make the threshold impossible to reach,
  # just return false
  # TODO: this is technically a less strict requirement than our actual cutoff
  if tagArr1.length == tagArr2.length == 0
    return true
  if tagArr1.length / tagArr2.length <= (1-EDIT_DIST_THRESHOLD) or
      tagArr2.length / tagArr1.length <= (1-EDIT_DIST_THRESHOLD)
    return false

  # just regular ol' levenshtein
  editDist = (arr1, arr2) ->
    memoizeMatrix = for a in arr1
      []
    for b in arr2
      for m in memoizeMatrix
        m.push(null)

    editDistRecurse = (i,j) ->
      if undefined != memoizeMatrix[i]?[j] != null
        return memoizeMatrix[i][j]
      else
        if i >= arr1.length
          return arr2.length - j
        if j >= arr2.length
          return arr1.length - i

        retDist = Math.min(
          editDistRecurse(i+1,j),
          editDistRecurse(i,j+1),
          editDistRecurse(i+1,j+1)
        )

        if arr1[i] != arr2[j]
          retDist += 1

        memoizeMatrix[i][j] = retDist
        return retDist
    return editDistRecurse(0,0)

  # normalized edit distance (divide by average of total length)
  return editDist(tagArr1, tagArr2)/((tagArr1.length+tagArr2.length)/2) <= EDIT_DIST_THRESHOLD


fs.readFile('lel.txt', 'utf8', (err, data) ->
  entries = for pageText in data.split('\n').filter((x) -> x.length > 0)
    new Node(JSON.parse(pageText).dom)

  JSON.stringify(templatizeNode(entries[0]).toDict())
  #console.log JSON.stringify(templatizeNode(entries[1]).toDict())
  #console.log JSON.stringify(templatizeNode(entries[2]).toDict())
)

