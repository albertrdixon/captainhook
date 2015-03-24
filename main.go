package main

import (
  "flag"
  log "github.com/Sirupsen/logrus"
  "net/http"
  "os"

  "github.com/gorilla/mux"
)

var (
  configdir  string
  echo       bool
  listenAddr string
  logLevel   int
  logFile    string
  auth       bool
)

func init() {
  flag.StringVar(&configdir, "configdir", "", "config dir to use")
  flag.BoolVar(&echo, "echo", false, "send output from script")
  flag.StringVar(&listenAddr, "listen-addr", "127.0.0.1:8080", "http listen address")
  flag.IntVar(&logLevel, "v", 1, "log level (0:quiet 1:info/default 2:debug)")
  flag.StringVar(&logFile, "log", "", "log file (default: STDOUT)")
  flag.BoolVar(&auth, "auth", true, "do basic authentication")
}

func main() {
  flag.Parse()
  if configdir == "" {
    os.Stderr.WriteString("configdir is required\n")
    os.Exit(1)
  }

  if logFile != "" {
    out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
      os.Stderr.WriteString("Could not open logfile, will use stdout: " + err.Error() + "\n")
      log.SetOutput(os.Stdout)
    } else {
      log.SetOutput(out)
    }
  }
  switch {
  case logLevel < 1:
    log.SetLevel(log.ErrorLevel)
  case logLevel == 1:
    log.SetLevel(log.InfoLevel)
  case logLevel > 1:
    log.SetLevel(log.DebugLevel)
  }

  r := mux.NewRouter()
  r.HandleFunc("/{id}", hookHandler).Methods("POST")
  http.Handle("/", r)

  log.WithFields(log.Fields{
    "listen":     listenAddr,
    "config-dir": configdir,
  }).Info("=== Booting CaptainHook, matey! Arr!")
  if err := http.ListenAndServe(listenAddr, nil); err != nil {
    log.WithField("error", err).Fatal("Server Error!")
  }
}
