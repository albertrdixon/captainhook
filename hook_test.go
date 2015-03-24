package main

import (
  "bytes"
  "fmt"
  log "github.com/Sirupsen/logrus"
  "io"
  "io/ioutil"
  "net/http"
  "net/http/httptest"
  "os"
  "path"
  "testing"

  "github.com/gorilla/mux"
)

var hookHandlerScript = `
{
  "scripts": [
    {
      "command": "echo",
      "args": [
        "foo"
      ]
    }
  ]
}`

var hookHandlerScriptDenied = `
{
  "scripts": [
    {
      "command": "echo",
      "args": [
        "foo"
      ]
    }
  ],
  "allowedNetworks": [
    "10.0.0.0/8"
  ]
}`

var hookResponseBody = `{
  "results": [
    {
      "stdout": "foo\n",
      "stderr": "",
      "status_code": 0
    }
  ]
}`

var data = []byte(`{"test": "test"}`)

var exposePostHandlerScript = `
{
  "scripts": [
    {
      "command": "cat"
    }
  ]
}
`

var exposePostResponseBody = `{
  "results": [
    {
      "stdout": "{\"Accept-Encoding\":\"gzip\",\"Authorization\":\"Basic Og==\",\"Content-Length\":\"16\",\"User-Agent\":\"Go 1.1 package http\"} {\"test\": \"test\"}",
      "stderr": "",
      "status_code": 0
    }
  ]
}`

var hookHanderTests = []struct {
  body       string
  echo       bool
  auth       bool
  token      string
  script     string
  statusCode int
  postBody   io.Reader
}{
  {"", false, false, "", hookHandlerScript, 200, nil},
  {"Not authorized.\n", false, false, "", hookHandlerScriptDenied, 401, nil},
  {"Not authorized.\n", false, true, "bad", hookHandlerScript, 401, nil},
  {hookResponseBody, true, false, "", hookHandlerScript, 200, nil},
  {hookResponseBody, true, true, "good", hookHandlerScript, 200, nil},
  {exposePostResponseBody, true, false, "", exposePostHandlerScript, 200, bytes.NewBuffer(data)},
}

func TestHookHandler(t *testing.T) {
  // Start a test server so we can test using the gorilla mux.
  r := mux.NewRouter()
  r.HandleFunc("/{id}", hookHandler).Methods("POST")
  ts := httptest.NewServer(r)
  defer ts.Close()

  log.SetLevel(log.ErrorLevel)
  os.Setenv("CPNHOOK_TOKEN", "good")

  // Set configdir option
  tempdir := os.TempDir()
  configdir = tempdir

  for _, tt := range hookHanderTests {
    // Set the echo config option.
    echo = tt.echo
    // Set the auth config option
    auth = tt.auth

    f, err := os.Create(path.Join(tempdir, "test.json"))
    if err != nil {
      t.Errorf(err.Error())
    }
    defer os.Remove(f.Name())
    defer f.Close()

    _, err = f.WriteString(tt.script)
    if err != nil {
      t.Errorf(err.Error())
    }

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", ts.URL, "test"), tt.postBody)
    req.SetBasicAuth(tt.token, "")
    if err != nil {
      t.Errorf(err.Error())
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
      t.Errorf(err.Error())
    }
    if resp.StatusCode != tt.statusCode {
      t.Errorf("wanted %d, got %d", tt.statusCode, resp.StatusCode)
    }

    data, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      t.Errorf(err.Error())
    }
    if string(data) != tt.body {
      t.Errorf("wanted %s, got %s", tt.body, string(data))
    }
  }
}
