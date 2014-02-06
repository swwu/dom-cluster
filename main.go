package main

import (
  "math/rand"
  "time"
  "fmt"
  "log"
  "io"
  "github.com/predictive-edge/dom-cluster/cluster"
  "github.com/predictive-edge/dom-cluster/dom"
  "os"
  "bufio"
  "encoding/json"
)

func getEntries(filename string) []*dom.Entry {
  f, err := os.Open(filename)
  if err != nil {
    log.Fatal(err)
  }
  defer f.Close()

  r := bufio.NewReaderSize(f,512*1024)
  //r.ReadLine() // extra read since first one is template
  line, isPrefix, err := r.ReadLine()
  entries := []*dom.Entry{}
  for err == nil && !isPrefix {
    var entry dom.Entry
    json.Unmarshal(line, &entry)

    if entry.Dom != nil {
      entries = append(entries, &entry)
    }
    line, isPrefix, err = r.ReadLine()
  }
  if err != io.EOF && err != nil {
      log.Fatal(err)
  }
  if isPrefix {
    log.Fatal("buffer size too small")
  }

  return entries
}

func main() {
  //defer profile.Start(profile.CPUProfile).Stop()
  rand.Seed(time.Now().UTC().UnixNano())

  entries := getEntries("./transformedrealdata.txt")
  //entry := entries[0]
  //templatizedNode := TemplatizeNode(&entry.Dom,10)
  //fmt.Println(templatizedNode)

  //wrappers := []*dom.Node{}
  for _,entry := range entries {
    //fmt.Println(entry.Uri)
    entry.Dom = cluster.NodeToWrapper(entry.Dom,10)
    //wrappers = append(wrappers, cluster.NodeToWrapper(&entry.Dom,10))
  }
  //fmt.Println("====")

  templates := cluster.DoCluster(entries)

  for _,t := range templates {
    fmt.Println(t.BaseUri)
    fmt.Println(t.Uris)
  }
  //for _,t1 := range wrappers {
  //  for _,t2 := range wrappers {
  //    _, score := cluster.NodeMerge(t1,t2)
  //    if score > 0.4 {
  //      fmt.Printf("0 ")
  //    } else {
  //      fmt.Printf("1 ")
  //    }
  //    //fmt.Printf("%f ",score)
  //  }
  //  fmt.Printf("\n")
  //}
  //a := wrappers[0]
  //b := wrappers[1]
  //merged, score := cluster.NodeMerge(a,b)
  //fmt.Println(merged.Children)
  //fmt.Println(score)
}
