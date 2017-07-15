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
	"html/template"
	"io"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path/filepath"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

var (
	socketPath = flag.String("sock", "", "FastCGI socket.")
	root       = flag.String("root", "/upload", "Path relative to root.")

	nameInvalid = regexp.MustCompile(`[^A-Za-z0-9._-]`)
	tmplRoot    = template.Must(template.New("root").Parse(`
<html>
<form enctype="multipart/form-data" action="{{.}}upload" method="post">
  <input type="file" name="file" accept="*/*" multiple />
  <input type="submit" />
</form>
</html>
`))

	tmplUpload = template.Must(template.New("root").Parse(`
<html>
Upload complete!
<a href="{{.}}">Back</a>
</html>
`))
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	tmplRoot.Execute(w, prefix(*root))
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

func main() {
	flag.Parse()
	m := mux.NewRouter()
	s := m.PathPrefix(prefix(*root)).Subrouter()
	s.Methods("GET").Subrouter().HandleFunc("/", handleRoot)
	s.Methods("POST").Subrouter().HandleFunc("/upload", handleUpload)

	os.Remove(*socketPath)
	sock, err := net.Listen("unix", *socketPath)
	if err != nil {
		log.Fatalf("Unable to listen to socket: %v", err)
	}
	if err := os.Chmod(*socketPath, 0666); err != nil {
		log.Fatal("Unable to chmod socket: ", err)
	}

	log.Printf("Running")
	log.Fatal(fcgi.Serve(sock, m))
}
