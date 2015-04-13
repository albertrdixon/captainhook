package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"

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
		return
	}
	h, err := json.Marshal(headers)
	if err != nil {
		return
	}
	i.Stdin = bytes.Join([][]byte{h, body}, []byte("\n"))
	return
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	log.WithFields(log.Fields{
		"hook":    id,
		"address": r.RemoteAddr,
	}).Info("Recieved webhook.")

	rb, err := NewRunBook(id)
	if err != nil {
		log.WithFields(log.Fields{
			"hook":  id,
			"error": err,
		}).Error("RunBook Error!")
		http.Error(w, err.Error(), 500)
		return
	}
	remoteIP := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	if !rb.AddrIsAllowed(remoteIP) {
		log.WithFields(log.Fields{
			"hook":    id,
			"address": r.RemoteAddr,
		}).Warn("Not Authorized!")
		http.Error(w, "Not authorized.", http.StatusUnauthorized)
		return
	}
	if !rb.Authorized(r) {
		log.WithFields(log.Fields{
			"hook":    id,
			"address": r.RemoteAddr,
		}).Warn("Authentication Failure!")
		http.Error(w, "Not authorized.", http.StatusUnauthorized)
		return
	}

	log.WithFields(log.Fields{
		"hook":    id,
		"address": r.RemoteAddr,
	}).Debug("Gathering request input.")
	in, err := gatherInput(r)
	if err != nil {
		log.WithFields(log.Fields{
			"hook":  id,
			"error": err,
		}).Error("Could not parse request!")
	}

	log.WithFields(log.Fields{
		"hook":        id,
		"address":     r.RemoteAddr,
		"num_scripts": len(rb.Scripts),
	}).Info("Executing hook scripts.")

	if rb.Async {
		go rb.execute(in)
		log.WithFields(log.Fields{
			"hook":        id,
			"address":     r.RemoteAddr,
			"num_scripts": len(rb.Scripts),
		}).Info("Script execution started, returning 200.")
		w.WriteHeader(200)
		return
	}

	response, err := rb.execute(in)
	if err != nil {
		log.WithFields(log.Fields{
			"hook":  id,
			"error": err,
		}).Error("Execute Error!")
		http.Error(w, err.Error(), 500)
		return
	}
	log.WithFields(log.Fields{
		"hook":    id,
		"address": r.RemoteAddr,
		"time":    rb.ExecTime,
	}).Info("Script execution complete.")

	if echo {
		log.WithFields(log.Fields{
			"hook":    id,
			"address": r.RemoteAddr,
		}).Info("Writing hook response.")
		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.WithFields(log.Fields{
				"hook":  id,
				"error": err,
			}).Error("Error generating response json!")
		}
		w.Write(data)
	}
}
