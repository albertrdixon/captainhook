package main

import (
  "encoding/json"
  log "github.com/Sirupsen/logrus"
  "net"
  "testing"
)

var allowedNetworksSuccessScript = `
{
  "scripts": [
    {
      "command": "echo"
    }
  ],
  "allowedNetworks": [
    "127.0.0.1/32",
    "10.0.0.0/8"
  ]
}`

var allowedNetworksFailureScript = `
{
  "scripts": [
    {
      "command": "echo"
    }
  ],
  "allowedNetworks": [
    "127.0.0.1/32",
    "10.0"
  ]
}`

var echoScript = script{Command: "cat"}

func TestNetworkUnmarshalling(t *testing.T) {
  log.SetLevel(log.ErrorLevel)

  r := runBook{}
  err := json.Unmarshal([]byte(allowedNetworksSuccessScript), &r)
  if err != nil {
    t.Errorf("JSON unmarshalling of allowed sources failed: %v", err)
  }
  if len(r.AllowedNetworks.Networks) != 2 {
    t.Errorf("JSON unmarshalling didn't produce the correct result: %v", r)
  }

  r = runBook{}
  err = json.Unmarshal([]byte(allowedNetworksFailureScript), &r)
  if err == nil {
    t.Errorf("JSON unmarshalling of allowed sources unexpectedly succeeded: %v", r)
  }
}

func TestAddrIsAllowed(t *testing.T) {
  log.SetLevel(log.ErrorLevel)

  testIPs := []struct {
    ip     net.IP
    result bool
  }{
    {net.ParseIP("127.0.0.1"), true},
    {net.ParseIP("172.16.0.1"), false},
    {net.ParseIP("10.0.0.1"), true},
    {net.ParseIP("10.0.1.1"), false},
  }

  nets := make([]net.IPNet, 2)
  for i, cidr := range []string{"127.0.0.1/32", "10.0.0.0/24"} {
    _, ipnet, _ := net.ParseCIDR(cidr)
    nets[i] = *ipnet
  }

  r := runBook{AllowedNetworks: Networks{Networks: nets}}

  for _, testIP := range testIPs {
    if r.AddrIsAllowed(testIP.ip) != testIP.result {
      t.Errorf("AddrIsAllowed %v expected %v", testIP.ip, testIP.result)
    }
  }

}

func TestInput(t *testing.T) {
  log.SetLevel(log.ErrorLevel)

  r := runBook{Scripts: []script{echoScript}}
  tests := []struct {
    in string
  }{
    {"123 456"},
    {"123\n456"},
    {"123"},
    {"123 456 789"},
  }

  for _, test := range tests {
    in := input{Stdin: []byte(test.in)}
    resp, err := r.execute(in)
    if err != nil {
      t.Errorf("runBook.execute(%q): Got error: %v", test.in, err)
    } else {
      out := resp.Results[0].Stdout
      if out != test.in {
        t.Errorf("runBook.execute(%q): Expected %q, got %q", test.in, test.in, out)
      }
    }
  }
}
