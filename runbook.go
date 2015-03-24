package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  log "github.com/Sirupsen/logrus"
  "io/ioutil"
  "net"
  "os/exec"
  "syscall"
  "time"
)

// runBook represents a collection of scripts.
type runBook struct {
  ID              string
  ExecTime        time.Duration
  Scripts         []script `json:"scripts"`
  AllowedNetworks Networks `json:"allowedNetworks,omitempty"`
}

type runBookResponse struct {
  Results []result `json:"results"`
}

type result struct {
  Stdout     string `json:"stdout"`
  Stderr     string `json:"stderr"`
  StatusCode int    `json:"status_code"`
}

type script struct {
  Command string   `json:"command"`
  Args    []string `json:"args"`
}

// Networks is its own struct for JSON unmarshalling gymnastics
type Networks struct {
  Networks []net.IPNet
}

// UnmarshalJSON for custom type Networks
func (nets *Networks) UnmarshalJSON(data []byte) error {
  ns := []string{}
  if err := json.Unmarshal(data, &ns); err != nil {
    return err
  }

  nets.Networks = make([]net.IPNet, len(ns))
  for i, nw := range ns {
    _, ipnet, err := net.ParseCIDR(nw)
    if err != nil {
      return err
    }
    nets.Networks[i] = *ipnet
  }
  return nil
}

// NewRunBook returns the runBook identified by id.
func NewRunBook(id string) (*runBook, error) {
  return getRunBookById(id)
}

func (r *runBook) AddrIsAllowed(remoteIP net.IP) bool {
  if len(r.AllowedNetworks.Networks) == 0 {
    return true
  }
  for _, nw := range r.AllowedNetworks.Networks {
    if nw.Contains(remoteIP) {
      return true
    }
  }
  return false
}

func (r *runBook) trackTime(start time.Time) {
  r.ExecTime = time.Since(start)
}

func (rb *runBook) execute(in input) (*runBookResponse, error) {
  defer rb.trackTime(time.Now())
  results := make([]result, 0)
  for _, x := range rb.Scripts {
    log.WithFields(log.Fields{
      "hook":   rb.ID,
      "script": x.Command,
    }).Debug("Executing script.")
    r, err := execScript(x, in)
    if err != nil {
      log.WithFields(log.Fields{
        "hook":   rb.ID,
        "script": x.Command,
        "error":  err,
      }).Errorf("Script failed! STDERR: %s", r.Stderr)
    }
    log.WithFields(log.Fields{
      "hook":   rb.ID,
      "script": x.Command,
    }).Debugf("Script results: %+v", r)
    results = append(results, r)
  }
  return &runBookResponse{results}, nil
}

func execScript(s script, in input) (r result, err error) {
  cmd := exec.Command(s.Command, s.Args...)
  log.WithField("script", s.Command).Debugf("Script: %+v", s)
  stdin, err := cmd.StdinPipe()
  if err != nil {
    log.WithFields(log.Fields{
      "script": s.Command,
      "error":  err,
    }).Error("Unable to create STDIN pipe!")
  }
  var stdout bytes.Buffer
  var stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr
  log.WithField("script", s.Command).Debugf("Writing STDIN: %s", in.Stdin)
  stdin.Write(in.Stdin)
  stdin.Close()
  err = cmd.Run()
  r.Stdout = stdout.String()
  r.Stderr = stderr.String()
  if err == nil {
    r.StatusCode = cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
  } else {
    r.StatusCode = -1
  }
  return
}

func getRunBookById(id string) (*runBook, error) {
  var r = new(runBook)
  r.ID = id
  runBookPath := fmt.Sprintf("%s/%s.json", configdir, id)
  data, err := ioutil.ReadFile(runBookPath)
  if err != nil {
    return r, fmt.Errorf("cannot read run book %s: %s", runBookPath, err)
  }
  err = json.Unmarshal(data, r)
  if err != nil {
    return r, err
  }
  return r, nil
}
