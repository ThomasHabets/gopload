// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path/filepath"
	"regexp"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	socketPath = flag.String("sock", "", "FastCGI socket.")
	listen     = flag.String("listen", ":8081", "Address to listen to.")
	root       = flag.String("root", "/upload", "Path relative to root.")
	out        = flag.String("out", ".", "Output directory.")

	nameInvalid = regexp.MustCompile(`[^A-Za-z0-9._-]`)
	upgrader    = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	tmplRoot.Execute(w, prefix(*root))
}

func handleUploadWS(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
		http.Error(w, "Failed to upgrade to websocket", http.StatusBadRequest)
		return
	}
	defer conn.Close()
	if err := func() error {
		name := nameInvalid.ReplaceAllString(mux.Vars(req)["filename"], "_")
		outName := filepath.Join(*out, name)
		ok := false
		for i := 0; i < 100; i++ {
			if _, err := os.Stat(outName); err != nil {
				ok = true
				break
			}
			outName = filepath.Join(*out, fmt.Sprintf("%d.%s", i, name))
		}
		if !ok {
			return fmt.Errorf("couldn't find a name for %q", name)
		}
		tmpName := outName + ".part"

		log.Printf("Uploading %q using websockets (tmpf %q, out %q)...", name, tmpName, outName)

		f, err := os.OpenFile(tmpName, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Couldn't open %q: %v", tmpName, err)
			return fmt.Errorf("couldn't open tempfile")
		}
		defer f.Close()
		defer os.Remove(tmpName)

		written := 0
		for {
			mt, p, err := conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			if mt != websocket.BinaryMessage {
				return fmt.Errorf("non-binary message received")
			}
			if err != nil {
				return fmt.Errorf("server receiving data: %v", err)
			}
			// log.Printf("Data received; %d bytes", len(p))
			if n, err := f.Write(p); err != nil {
				return fmt.Errorf("writing to file: %v", err)
			} else if n < len(p) {
				return fmt.Errorf("short write to file: %v", err)
			}
			written += len(p)
			if err := conn.WriteJSON(&struct{ Written int }{Written: written}); err != nil {
				log.Printf("Failed to send status message: %v", err)
			}
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("closing file: %v", err)
		}
		if err := os.Rename(tmpName, outName); err != nil {
			return fmt.Errorf("rename temp file to final file: %v", err)
		}
		return nil
	}(); err != nil {
		log.Printf("Upload failed: %v", err)
		if err := conn.WriteJSON(&struct{ Error string }{Error: err.Error()}); err != nil {
			log.Printf("Failed to send error message: %v", err)
		}
		return
	}
	log.Printf("Done uploading...")
}

func handleUpload(w http.ResponseWriter, req *http.Request) {
	r, err := req.MultipartReader()
	if err != nil {
		log.Error(err)
		return
	}
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error! %v", err)
			return
		}
		name := nameInvalid.ReplaceAllString(part.FileName(), "_")
		log.Printf("Uploading %q", name)
		if err := func() error {
			defer part.Close()
			out, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, part); err != nil {
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			log.Printf("Error! %v", err)
			return
		}
	}
	log.Printf("Upload complete!")
	tmplUpload.Execute(w, prefix(*root))
}

func prefix(r string) string {
	r = filepath.Clean(r)
	if r == "/" {
		return r
	}
	return r + "/"
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 bitch for path %q\n", r.URL)
}

func main() {
	flag.Parse()
	m := mux.NewRouter()
	m.NotFoundHandler = http.HandlerFunc(notFound)

	s := m.PathPrefix(prefix(*root)).Subrouter()
	s.Methods("GET").Subrouter().HandleFunc("/", handleRoot)
	s.Methods("POST").Subrouter().HandleFunc("/upload", handleUpload)
	s.HandleFunc("/upload-ws/{filename}", handleUploadWS)

	if *socketPath != "" {
		os.Remove(*socketPath)
		sock, err := net.Listen("unix", *socketPath)
		if err != nil {
			log.Fatalf("Unable to listen to socket %q: %v", *socketPath, err)
		}
		if err := os.Chmod(*socketPath, 0666); err != nil {
			log.Fatal("Unable to chmod socket: ", err)
		}
		log.Printf("Running")
		log.Fatal(fcgi.Serve(sock, m))
	} else {
		s := &http.Server{
			Addr:           *listen,
			Handler:        m,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		log.Printf("Running")
		log.Fatal(s.ListenAndServe())
	}

}
