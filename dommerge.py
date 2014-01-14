import uuid
import math
import copy

import json

# weight for structural node edit-distance
DIFF_NODE_DIST = 100
# weight for text-node content edit-distance
DIFF_TEXT_DIST = 1
# damping coefficient per level of the DOM tree
LEVEL_DIST_MULTIPLIER = 1/math.e


# token used to concatenate memoize_tokens for memoization of n-arg fns
MEMOIZE_CONCAT_TOKEN = "|||"


def read_or_none(dictObj, field):
  return dictObj[field] if (dictObj and field in dictObj) else None

class Node(object):

  nodeName = None
  tagName = None
  attrs = None
  text = None

  children = None

  editWeight = 0

  def __init__(self, nodeDict = None):
    # filler nodes are those that differ between two merged nodes
    self.is_filler = False

    # memoize_token is used to memoize node operations
    self.memoize_token = uuid.uuid4()

    if not nodeDict: nodeDict = {}
    # things that describe the node itself
    self.nodeName = read_or_none(nodeDict, "nodeName")
    self.tagName = read_or_none(nodeDict, "tagName")
    self.attrs = read_or_none(nodeDict, "attrs")
    self.text = read_or_none(nodeDict, "text")
    if "children" in nodeDict:
      self.children = [Node(node) for node in read_or_none(nodeDict, "children")]
    else:
      self.children = []

  def getText(self):
    if self.nodeName == "#text":
      if self.is_filler == False:
        return self.text
    return None

  # the total "edit distance" penalty for inserting/deleting this node and all its children
  def getEditWeight(self):
    if self.editWeight == 0:
      self.editWeight += (LEVEL_DIST_MULTIPLIER *
          sum(c.getEditWeight() for c in self.children)) if self.children else 0
      self.editWeight +=  DIFF_TEXT_DIST if self.nodeName == "#text" else DIFF_NODE_DIST
    return self.editWeight

  def makeCopy(self):
    retCopy = copy.copy(self)
    retCopy.editWeight = 0
    return retCopy

  def toDict(self):
    return {
        "nodeName": self.nodeName,
        "tagName": self.tagName,
        "attrs": self.attrs,
        "text": self.text,
        "children": [c.toDict() for c in (self.children or [])]
        }
  def toJson(self):
    return json.dumps(self.toDict())

# TextFillerNode indicates that text has changed but structure has not
class TextFillerNode(Node):
  def __init__(self):
    self.is_filler = True
    self.memoize_token = uuid.uuid4()

    self.nodeName = "#text"
    self.tagName = None

# DomFillerNode indicates that structure has changed
class DomFillerNode(Node):
  def __init__(self):
    self.is_filler = True
    self.memoize_token = uuid.uuid4()

    self.nodeName = "#filler"
    self.tagName = None

# memoize a function with prototype (Node, Node)
def memoizeNodes(nodeFn):
  memoObj = {}
  def retFn(*args):
    memoTokens = []
    for node in args:
      memoTokens.append(node.memoize_token)
    memoTokens.sort()
    memoToken = MEMOIZE_CONCAT_TOKEN.join(
        str(token) for token in memoTokens)
    if memoToken in memoObj:
      return memoObj[memoToken]
    else:
      retVal = nodeFn(*args)
      memoObj[memoToken] = retVal
      return retVal
  return retFn

def nodeMerge(node1, node2):
  return mergeNodes(node1, node2)[0]

def nodeDist(node1, node2):
  return mergeNodes(node1, node2)[1] / (node1.getEditWeight() +
      node2.getEditWeight())

# returns (merged_node, edit_dist) pair
def mergeNodes(node1, node2):
  # if nodeNames differ, return structure-filler
  if node1.nodeName != node2.nodeName:
    return (DomFillerNode(), node1.getEditWeight() + node2.getEditWeight())
  # guaranteed that node1.nodeName == node2.nodeName from this point on

  # if both fillers, just return a filler of the appropriate type
  if node1.is_filler and node2.is_filler:
    if node1.nodeName == "#text":
      return (TextFillerNode(), 0)
    else:
      return (DomFillerNode(), 0)

  # if both non-filler text nodes,
  # otherwise just return the text node
  if node1.nodeName == "#text":
    # generate a filler text node if text differs
    if node1.getText() != node2.getText():
      return (TextFillerNode(), 2*DIFF_TEXT_DIST)
    # return the (shallow-copied) same text node if text doesn't differ
    else:
      return (node1.makeCopy(), 0)

  # otherwise we will keep this node but merge its children
  retNode = node1.makeCopy()
  retNode.children = None
  retDist = 0

  # if either node has children, merge children and get scores
  n1c = node1.children or []
  n2c = node2.children or []
  if len(n1c) == 0 or len(n2c) == 0:
    retNode.children = []
  else:
    # zip(*arr) unzips arr
    mergedNodes, mergedDists = zip(*[mergeNodes(c1,c2) for c1,c2 in zip(n1c, n2c)])

    retNode.children = mergedNodes
    retDist += sum(mergedDists)
    retDist += (sum(c.getEditWeight() for c in n1c[len(mergedNodes):])
        + sum(c.getEditWeight() for c in n1c[len(mergedNodes):]))
  retDist *= LEVEL_DIST_MULTIPLIER

  return (retNode, retDist)

mergeNodes = memoizeNodes(mergeNodes)
