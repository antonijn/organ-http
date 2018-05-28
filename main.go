package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
)

var recordingsDir string

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintln(w, "<html><head><title>Johannus Dashboard</title></head><body>")
		fmt.Fprintln(w, "<h1>Johannus Organ Dashboard</h1>")

		files, err := ioutil.ReadDir(recordingsDir)
		if err != nil {
			fmt.Fprintln(w, "<p>Error: recordings inaccessible</p>")
		} else {
			fmt.Fprintln(w, "<h2>Recordings:</h2>")
			fmt.Fprintln(w, "<table><tr><th>Download</th><th>Size (MiB)</th><th>Delete</th></tr>")

			for _, file := range files {
				filename := file.Name()
				filePath := path.Join("/audio/", filename)
				fileUrl, _ := url.Parse(filePath)
				fmt.Fprintln(w, "<tr>")
				fmt.Fprintf(w, "<td><a href=%s>%s</a></td><td>%.1f</td>", fileUrl, html.EscapeString(filename), float32(file.Size()) / (1024 * 1024))
				fmt.Fprintf(w, "<td><form method=\"post\" onsubmit=\"return confirm('Are you sure?');\"><button type=\"submit\" name=\"deleterecording\" value=%s>delete</button></td>", html.EscapeString(filename))
				fmt.Fprintln(w, "</tr>")
			}

			fmt.Fprintln(w, "</table>")
		}
		fmt.Fprintln(w, "</body></html>")
		break

	case "POST":
		defer http.Redirect(w, r, "/", http.StatusSeeOther)
		r.ParseForm()
		delRec := r.Form.Get("deleterecording")
		if delRec == "" {
			log.Println("Unhandled POST request")
			return
		}

		if path.Base(delRec) != delRec {
			log.Println("Invalid resource delete attempt")
			return
		}

		recPath := path.Join(recordingsDir, delRec)

		if err := os.Remove(recPath); err != nil {
			log.Print(err)
			return
		}

		break
	}
}

func main() {
	home := os.Getenv("HOME")
	gorguepath := path.Join(home, "GrandOrgue")
	recordingsDir = path.Join(gorguepath, "Audio recordings")

	http.HandleFunc("/", handler)
	fserv := http.FileServer(http.Dir(recordingsDir))
	http.Handle("/audio/", http.StripPrefix("/audio", fserv))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
