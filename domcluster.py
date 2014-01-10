import json
import dommerge

import numpy as np
import pylab

pages = []
distance = {}

with open('lel.txt') as rf:
  pages = [dommerge.Node(json.loads(line)["dom"]) for line in rf]

for i,page1 in enumerate(pages):
  for j,page2 in enumerate(pages):
    if j <= i: continue
    distance[(i,j)] = dommerge.nodeDist(page1,page2)


members = list(distance.iteritems())
print members
members.sort(key=lambda x: x[1])
print members


