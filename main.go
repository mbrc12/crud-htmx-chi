package main

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	tmpl *template.Template
)

type Item struct {
	Id    int
	Entry string
}

func init() {
	godotenv.Load()
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
	log.Info("Loaded templates")
}

func main() {
	PORT := os.Getenv("PORT")
	DB_NAME := os.Getenv("DB_NAME")

	// open db
	var err error
	db, err = sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal(err)
		log.Fatalf("Failed to open db %s", DB_NAME)
	}
	defer db.Close()

	router := chi.NewRouter()

	// Use charmbracelet/log as the logger for chi
	router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger: &charmLogger{},
	}))

	// recover from panic
	router.Use(middleware.Recoverer)

	// gzip compression
	router.Use(middleware.Compress(5, "text/html"))

	// handle static files in ./static
	staticServer := http.FileServer(http.Dir("./static"))
	router.Handle("/static/*", http.StripPrefix("/static", staticServer))

	// main page
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		tmpl.ExecuteTemplate(w, "index", nil)
	})

	// initial load
	router.Get("/load", func(w http.ResponseWriter, r *http.Request) {
		render(r.Context(), w)
	})

	// add item
	router.Get("/add", func(w http.ResponseWriter, r *http.Request) {
		entry := r.FormValue("entry")
		ctx := r.Context()

		db.ExecContext(ctx, "insert into items (entry) values (?)", entry)

		if err := render(ctx, w); err != nil {
			log.Fatal(err)
		}
	})

	// delete item
	router.Get("/delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()

		db.ExecContext(ctx, "delete from items where id = ?", id)

		if err := render(ctx, w); err != nil {
			log.Fatal(err)
		}
	})

	// edit item
	router.Get("/edit/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		newEntry := r.FormValue("editing")

		ctx := r.Context()

		db.ExecContext(ctx, "update items set entry = ? where id = ?", newEntry, id)

		if err := render(ctx, w); err != nil {
			log.Fatal(err)
		}
	})

	http.ListenAndServe(PORT, router)
}

// fetch data from sql and render into html into the writer
func render(ctx context.Context, w http.ResponseWriter) error {
	rows, err := db.QueryContext(ctx, "select id, entry from items")
	if err != nil {
		return err
	}

	var data []Item

	for rows.Next() {
		var item Item
		rows.Scan(&item.Id, &item.Entry)
		data = append(data, item)
	}

	w.Header().Set("Content-Type", "text/html")
	return tmpl.ExecuteTemplate(w, "list", data)
}

type charmLogger struct{}

func (self *charmLogger) Print(v ...interface{}) {
	log.Infof("chi: %v", v)
}
