package main

import (
  "bytes"
  "encoding/json"
  "io"
  "io/ioutil"
  "log"
  "net"
  "net/http"
  "strings"

  "github.com/gorilla/mux"
)

type input struct {
  Stdin []byte
}

func gatherInput(r *http.Request) (i input, err error) {
  headers := make(map[string]string, len(r.Header))
  for k := range r.Header {
    headers[k] = r.Header.Get(k)
  }
  body, err := ioutil.ReadAll(r.Body)
  if err != nil && err != io.EOF {
    log.Fatal(err)
    return
  }
  h, err := json.Marshal(headers)
  if err != nil {
    log.Fatal(err)
    return
  }
  i.Stdin = bytes.Join([][]byte{h, body}, []byte("\n"))
  return
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
  params := mux.Vars(r)
  id := params["id"]
  log.Printf("Received hook for id '%s' from %s\n", id, r.RemoteAddr)
  rb, err := NewRunBook(id)
  if err != nil {
    log.Println(err.Error())
    http.Error(w, err.Error(), 500)
    return
  }
  remoteIP := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
  if !rb.AddrIsAllowed(remoteIP) {
    log.Printf("Hook id '%s' is not allowed from %v\n", id, r.RemoteAddr)
    http.Error(w, "Not authorized.", http.StatusUnauthorized)
    return
  }
  in, err := gatherInput(r)
  response, err := rb.execute(in)
  if err != nil {
    log.Println(err.Error())
    http.Error(w, err.Error(), 500)
    return
  }
  if echo {
    data, err := json.MarshalIndent(response, "", "  ")
    if err != nil {
      log.Println(err.Error())
    }
    w.Write(data)
  }
}

func interoplatePOSTData(rb *runBook, r *http.Request) {
  if r.ContentLength == 0 {
    return
  }
  data, err := ioutil.ReadAll(r.Body)
  if err != nil && err != io.EOF {
    log.Fatal(err)
    return
  }
  defer r.Body.Close()
  stringData := string(data[:r.ContentLength])
  for i := range rb.Scripts {
    for j := range rb.Scripts[i].Args {
      rb.Scripts[i].Args[j] = strings.Replace(rb.Scripts[i].Args[j], "{{POST}}", stringData, -1)
    }
  }
}
