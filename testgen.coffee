
# X = -log(U)/a is exponentially distributed with rate parameter a if U ~ uniform
# E(X) = 1/a
expRand = (rate) ->
  return -Math.log(Math.random()) / rate

randSelect = (arr) ->
  arr[Math.floor(Math.random() * arr.length)]

bernoulliChance = (chance) ->
  if Math.random() < chance
    return true
  return false

categorical = (chances, fns) ->
  rnd = Math.random() * chances.reduce((a,b) ->
    a+b
  , 0)
  count = 0

  for chance,i in chances
    count += chance
    if rnd < count
      return fns[i]()

randomSign = ->
  categorical([0.65,0.2,0.05,0.05,0.05], [
    ->
      1
    , ->
      Math.ceil(expRand(0.7))
    , ->
      '?'
    , ->
      '*'
    , ->
      '+'
  ])

newBlankDomNode = (leaf) ->
  possibleNodes = ["div","span","h1","p","ul","li"]
  leafOnlyNodes = ["#text"]
  if leaf
    nodeName = randSelect(possibleNodes.concat(leafOnlyNodes))
    if nodeName[0] == "#"
      tagName = undefined
    else
      tagName = nodeName
  else
    nodeName = randSelect(possibleNodes)
    tagName = nodeName
  return {
    nodeName: nodeName
    tagName: tagName
    attrs: {}
    children: []
    sign: 0
  }

generateParenNode = (maxDepth) ->
  paren = {
    nodeName: "##paren"
    tagName: undefined
    attrs: {}
    children: for i in [0...expRand(0.5)]
      generateTemplate(maxDepth-1)
    sign: randomSign()
  }

# generate a random template, with at most maxDepth layers of children
generateTemplate = (maxDepth) ->
  root = newBlankDomNode(maxDepth == 0)
  root.sign = randomSign()

  if maxDepth >= 1
    root.children = for i in [0...expRand(0.5)]
      categorical([0.9,0.1],[
        ->
          generateTemplate(maxDepth-1)
        ,->
          generateParenNode(maxDepth)
      ])

  return root

# create a node with the same nodename/tagname/attrs as another node
newNodeCopy = (template) ->
  return {
    nodeName: template.nodeName
    tagName: template.tagName
    attrs: template.attrs
    children: []
  }

generateFromTemplate = (template) ->
  curNode = newNodeCopy(template)

  curNode.children = []
  for c,i in template.children
    if c.sign == 0 or c.sign == 1
      curNode.children.push(generateFromTemplate(c))
    else if c.sign == '?' and bernoulliChance(0.5)
      curNode.children.push(generateFromTemplate(c))
    else if c.sign == '*' and bernoulliChance(0.5)
      for i in [0...expRand(0.5)]
        curNode.children.push(generateFromTemplate(c))
    else if c.sign == '+'
      for i in [0...expRand(0.5)]
        curNode.children.push(generateFromTemplate(c))
    else
      for i in [0...c.sign]
        curNode.children.push(generateFromTemplate(c))

  return curNode

generateEntryFromTemplate = (template) ->
  return {
    url: "http://example.com"
    dom: generateFromTemplate(template)
  }



template = generateTemplate(3)

console.log JSON.stringify(template)
for i in [0...100]
  console.log JSON.stringify(generateEntryFromTemplate(template))




