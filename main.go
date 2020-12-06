package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
)

type Configuration struct {
	RecordingsDir string `json:"recordings-directory"`
	OrganName     string `json:"organ-name"`
}

func readJsonConfig(fileName string, res *Configuration) error {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		return err
	}

	defer jsonFile.Close()

	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, res)
}

var appConfig Configuration = Configuration{"", "Custom Digital Organ"}

func getAppConfig() {
	globalConfigPath := path.Join("/", "etc", "organ-http", "config.json")
	localConfigPath := path.Join(os.Getenv("HOME"), ".config", "organ-http", "config.json")

	err := readJsonConfig(globalConfigPath, &appConfig)
	if err != nil {
		log.Print(err)
	}
	err = readJsonConfig(localConfigPath, &appConfig)
	if err != nil {
		log.Print(err)
	}
}

const javaScriptRenameCode = `
function showRenamePrompt(filename) {
	newname = prompt('Enter new file name', filename);
	if (filename != null) {
		document.getElementById(filename + '-newname').value = newname;
		return true;
	}

	return false;
}`

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		title := html.EscapeString(appConfig.OrganName)

		fmt.Fprintln(w, "<html>")
		fmt.Fprintln(w, "<head>")
		fmt.Fprintf(w, "<title>%s</title>\n", title)
		fmt.Fprintln(w, "<script>")
		fmt.Fprintln(w, javaScriptRenameCode)
		fmt.Fprintln(w, "</script>")
		fmt.Fprintln(w, "</head>")
		fmt.Fprintln(w, "<body>")
		fmt.Fprintf(w, "<h1>%s</h1>\n", title)

		files, err := ioutil.ReadDir(appConfig.RecordingsDir)
		if err != nil {
			fmt.Fprintln(w, "<p>Error: recordings inaccessible</p>")
		} else {
			sort.Slice(files, func(i,j int) bool{ return files[i].ModTime().After(files[j].ModTime()) })

			fmt.Fprintln(w, "<h2>Recordings</h2>")
			fmt.Fprintln(w, "<table><tr><th>Download</th><th>Size (MiB)</th><th>Rename</th><th>Delete</th></tr>")

			for _, file := range files {
				filename := file.Name()
				escFileName := html.EscapeString(filename)
				fmt.Fprintln(w, "<tr>")

				fmt.Fprintf(w, "<td><a href=\"/audio/%s\">%s</a></td>\n", escFileName, escFileName)
				fmt.Fprintf(w, "<td>%.1f</td>\n", float32(file.Size()) / (1024 * 1024))

				fmt.Fprintln(w, "<td>")
				fmt.Fprintln(w, "<form method=\"post\" onsubmit=\"return confirm('Are you sure?');\">")
				fmt.Fprintf(w, "<button type=\"submit\" name=\"deleterecording\" value=\"%s\">delete</button>\n", escFileName)
				fmt.Fprintln(w, "</form>")
				fmt.Fprintln(w, "</td>")

				fmt.Fprintln(w, "<td>")
				fmt.Fprintln(w, "<form method=\"post\">")
				fmt.Fprintf(w, "<input type=\"hidden\" name=\"newname\" id=\"%s-newname\">\n", escFileName)
				fmt.Fprintf(w, "<button type=\"submit\" name=\"renamerecording\" value=\"%s\" onClick=\"return showRenamePrompt('%s')\">rename</button>\n", escFileName, escFileName)
				fmt.Fprintln(w, "</form>")
				fmt.Fprintln(w, "</td>")

				fmt.Fprintln(w, "</tr>")
			}

			fmt.Fprintln(w, "</table>")
		}
		fmt.Fprintln(w, "</html>")
		fmt.Fprintln(w, "</body>")
		break

	case "POST":
		defer http.Redirect(w, r, "/", http.StatusSeeOther)
		r.ParseForm()
		delRec := r.Form.Get("deleterecording")
		if delRec != "" {
			if path.Base(delRec) != delRec {
				log.Println("Invalid resource delete attempt")
				return
			}

			recPath := path.Join(appConfig.RecordingsDir, delRec)

			if err := os.Remove(recPath); err != nil {
				log.Print(err)
				return
			}

			break
		}

		renameRec := r.Form.Get("renamerecording")
		if renameRec != "" {
			newname := r.Form.Get("newname")
			if path.Base(renameRec) != renameRec || path.Base(newname) != newname {
				log.Println("Invalid resource rename attempt")
				return
			}

			oldPath := path.Join(appConfig.RecordingsDir, renameRec)
			newPath := path.Join(appConfig.RecordingsDir, newname)
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Print(err)
				return
			}

			break
		}

		log.Println("Unhandled POST request")
		break
	}
}

func main() {
	getAppConfig()

	log.Printf("Starting file server in directory `%s'\n", appConfig.RecordingsDir)

	http.HandleFunc("/", handler)
	fserv := http.FileServer(http.Dir(appConfig.RecordingsDir))
	http.Handle("/audio/", http.StripPrefix("/audio", fserv))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
