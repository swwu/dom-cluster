merge = require('./dommerge')

fs = require 'fs'

CLUSTER_CUTOFF = 0.2

singleCluster = (pages, clustered) ->
  clusterUri = null
  for uri,page of pages
    if uri not of clustered
      clusterUri = uri
      break

  if clusterUri == null then return null
  clustered[clusterUri] = true
  clusterPage = pages[clusterUri]
  curCluster = [clusterUri]

  for uri,opage of pages
    if uri == clusterUri then continue
    if merge.nodeDist(clusterPage,opage) <= CLUSTER_CUTOFF
      curCluster.push(uri)
      clustered[uri] = true
  return curCluster


fs.readFile('lel.txt', 'utf8', (err, data) ->
  pages = Object.create(null)
  for pageText in data.split('\n').filter((x) -> x.length > 0)
    obj = JSON.parse(pageText)
    pages[obj.url] = new merge.Node(obj.dom)

  clustered = Object.create(null)
  clusters = []

  while Object.keys(clustered).length < Object.keys(pages).length
    clusters.push(singleCluster(pages, clustered))
  for cluster in clusters
    console.log cluster
)

