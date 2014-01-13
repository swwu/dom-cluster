import dommerge

import json

import numpy as np
import pylab

CLUSTER_CUTOFF = 0.2

def singleCluster(pages, clustered):
  clusterPage = None
  clusterPageUri = ""
  for uri,page in pages.iteritems():
    if uri not in clustered:
      clusterPage = page
      clusterPageUri = uri
      break
  if clusterPage == None: return None
  curCluster = set([clusterPageUri])
  for uri,opage in pages.iteritems():
    if dommerge.nodeDist(clusterPage, opage) <= CLUSTER_CUTOFF:
      curCluster.add(uri)
      clustered.add(uri)
  return curCluster



pages = {}
distance = {}

with open('lel.txt') as rf:
  for line in rf:
    obj = json.loads(line)
    pages[obj["url"]] = dommerge.Node(obj["dom"])

clustered = set()
clusters = []

while len(clustered) < len(pages):
  clusters.append(singleCluster(pages, clustered))

for cluster in clusters:
  print cluster


