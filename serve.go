package serve

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

// CLI runs the serve command line app and returns its exit status.
func CLI(args []string) int {
	var app app
	if err := app.fromArgs(args); err != nil {
		return 2
	}
	if err := app.run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

type app struct {
	dir   string
	quiet bool
	addr  string
}

func (app *app) fromArgs(args []string) error {
	flags := flag.NewFlagSet("hostr", flag.ContinueOnError)
	flags.StringVar(&app.addr, "a", "localhost:9876", "http address")
	flags.BoolVar(&app.quiet, "q", false, "use quiet mode - don't display logs")
	if err := flags.Parse(args); err != nil {
		return err
	}

	app.dir = "."
	fArgs := flags.Args()
	if len(fArgs) > 0 {
		app.dir = fArgs[0]
	}

	return nil
}

func (app *app) run() error {
	return http.ListenAndServe(app.addr, app.handler())
}

func (app *app) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.quiet {
			log.Printf("[%s] %s\n", r.Method, r.URL.Path)
		}

		w.Header().Set("Cache-Control", "no-store")

		switch r.Method {
		case http.MethodGet:
			if err := app.handleGet(w, r); err != nil {
				log.Println("Error:", err)
				http.Error(w, "File not found", http.StatusNotFound)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (app *app) handleGet(w http.ResponseWriter, r *http.Request) error {
	urlPath := filepath.Clean(r.URL.Path)
	dirPath := filepath.Join(app.dir, urlPath)

	f, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return err
		}
		serveDir(w, urlPath, files)

		return nil
	}

	_, err = io.Copy(w, f)
	return err
}

type file struct {
	Path    string
	Name    string
	Size    int64
	ModTime string
	IsDir   bool
}

const listTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" href=data:, />
    <title>Index of {{.Path}}</title>
    <style>
		* { font-family: monospace; }
		body { display: flex; flex-direction: column; align-items: center; }
		th { text-align: left; }
		th:before { content: ''; display: block; min-width: 75px; }
    </style>
</head>
<body>
	<h1>Index of {{.Path}}</h1>
	<table>
		<tr>
			<th>Name</th>
			<th>Size</th>
			<th>Last modified</th>
		</tr>
	{{range .Files}}
		<tr>
			<td><a href="{{ .Path }}">{{ .Name }}</a></td>
			<td>{{if not .IsDir}} {{ .Size }} {{end}}</td>
			<td>{{if not .IsDir}} {{ .ModTime }} {{end}}</td>
		</tr>
	{{end}}
	</table>
</body>
</html>`

func serveDir(w io.Writer, path string, files []os.FileInfo) error {
	t := template.Must(template.New("dirlist").Parse(listTemplate))

	fs := []file{}
	for _, f := range files {
		fs = append(fs, file{
			Name:    f.Name(),
			Path:    filepath.Join(path, f.Name()),
			Size:    f.Size(),
			ModTime: f.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   f.IsDir(),
		})
	}

	return t.Execute(w, struct {
		Path  string
		Files []file
	}{
		Path:  path,
		Files: fs,
	})
}
