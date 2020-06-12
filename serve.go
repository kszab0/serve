package serve

import (
	"archive/zip"
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
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
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
		case http.MethodPost:
			if err := app.handlePost(w, r); err != nil {
				log.Println("Error:", err)
				http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
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
		th:not(:first-child):before { content: ''; display: block; min-width: 75px; }
		table { margin-bottom: 16px; }
    </style>
</head>
<body>
	<h1>Index of {{.Path}}</h1>
	<form method="POST">
		<table>
			<tr>
				<th />
				<th>Name</th>
				<th>Size</th>
				<th>Last modified</th>
			</tr>
		{{range .Files}}
			<tr>
				<td><input type="checkbox" name="files" value="{{.Name}}" /></td>
				<td><a href="{{ .Path }}">{{ .Name }}</a></td>
				<td>{{if not .IsDir}} {{ .Size }} {{end}}</td>
				<td>{{if not .IsDir}} {{ .ModTime }} {{end}}</td>
			</tr>
		{{end}}
		</table>

		<input type="submit" value="Download zip" />
	</form>
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

func (app *app) handlePost(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	dir := filepath.Join(app.dir, filepath.Clean(r.URL.Path))

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, dirname(dir)))

	return archive(w, dir, r.Form["files"])
}

func dirname(dir string) string {
	base := filepath.Base(dir)
	if base == "." || base == "/" {
		base = "root"
	}
	return base
}

func archive(w io.Writer, dir string, filenames []string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	fileNames := filenames
	if len(fileNames) == 0 {
		if err := addToArchive(zw, dir, dir); err != nil {
			return err
		}
	}

	for _, name := range fileNames {
		if err := addToArchive(zw, dir, filepath.Join(dir, name)); err != nil {
			return err
		}
	}

	return nil
}

func addToArchive(zw *zip.Writer, dir, path string) error {
	relPath, err := filepath.Rel(dir, path)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if fi.IsDir() {
		fis, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		for _, fi := range fis {
			if err := addToArchive(zw, dir, filepath.Join(path, fi.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	zf, err := zw.Create(relPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(zf, f)
	return err
}
