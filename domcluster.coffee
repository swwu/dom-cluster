merge = require('./dommerge')

fs = require 'fs'

CLUSTER_CUTOFF = 0.4

singleCluster = (pages, clustered) ->
  clusterUri = null
  for uri,page of pages
    if uri not of clustered
      clusterUri = uri
      break

  if clusterUri == null then return null
  clustered[clusterUri] = true
  clusterPage = pages[clusterUri]
  curCluster = {
    seed: clusterUri
    others: Object.create(null)
  }

  for uri,opage of pages
    if uri == clusterUri then continue
    dist = merge.nodeDist(clusterPage,opage)
    if dist <= CLUSTER_CUTOFF
      curCluster.others[uri] = dist
      clustered[uri] = true
  return curCluster


fs.readFile('realdata.txt', 'utf8', (err, data) ->
  pages = Object.create(null)
  for pageText in data.split('\n').filter((x) -> x.length > 0)
    try
      obj = JSON.parse(pageText)
      pages[obj.request.uri] = new merge.Node(obj.dom)
    catch error
      console.log error

  clustered = Object.create(null)
  clusters = []

  while Object.keys(clustered).length < Object.keys(pages).length
    clusters.push(singleCluster(pages, clustered))
  for cluster in clusters
    console.log cluster

  a = 'http://mikesbikes.com/product-list/all-mountain-full-suspension-pg824/'
  b = 'http://mikesbikes.com/product/13raleigh-cadent-i2x8-174054-1.htm'
  c = 'http://mikesbikes.com/product/13raleigh-revenio-carbon-1.0-174491-1.htm'

  #console.log pages[c]
  #console.log JSON.stringify(merge.nodeMerge(pages[a],pages[b]).toDict())
  #console.log JSON.stringify(pages[b].toDict())
)

