package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"os"
	"html/template"
	_ "modernc.org/sqlite"
	"net/http"
)

type Object struct {
	Id        int64          //   ID INTEGER PRIMARY KEY AUTOINCREMENT
	Object_id string         // OBJECT_ID TEXT UNIQUE NOT NULL
	Parent_id string         // PARENT_ID TEXT NOT NULL
	Ref_id    sql.NullString // REF_ID TEXT DEFAULT NULL
	Class     string         // CLASS TEXT NOT NULL
	Detail_id int64          // DETAIL_ID INTEGER DEFAULT NULL
	Name      string         // NAME TEXT DEFAULT NULL
}

type Detail struct {
	Id         int64          // ID INTEGER PRIMARY KEY AUTOINCREMENT
	Path       sql.NullString // PATH TEXT DEFAULT NULL
	Size       sql.NullInt64  // SIZE INTEGER
	Timestamp  sql.NullInt64  // TIMESTAMP INTEGER
	Title      sql.NullString // TITLE TEXT COLLATE NOCASE
	Duration   sql.NullString // DURATION TEXT
	Bitrate    sql.NullInt64  // BITRATE INTEGER
	Samplerate sql.NullInt64  // SAMPLERATE INTEGER
	Creator    sql.NullString // CREATOR TEXT COLLATE NOCASE
	Artist     sql.NullString // ARTIST TEXT COLLATE NOCASE
	Album      sql.NullString // ALBUM TEXT COLLATE NOCASE
	Genre      sql.NullString // GENRE TEXT COLLATE NOCASE
	Comment    sql.NullString // COMMENT TEXT
	Channels   sql.NullInt64  // CHANNELS INTEGER
	Disc       sql.NullInt64  // DISC INTEGER
	Track      sql.NullInt64  // TRACK INTEGER
	// date sql.NullTime // DATE DATE
	Date       sql.NullString // DATE DATE
	Resolution sql.NullString // RESOLUTION TEXT
	Thumbnail  bool           // THUMBNAIL BOOL DEFAULT 0
	Album_art  sql.NullInt64  // ALBUM_ART INTEGER DEFAULT 0
	Rotation   sql.NullInt64  // ROTATION INTEGER
	Dlna_pn    sql.NullString // DLNA_PN TEXT
	Mime       sql.NullString // MIME TEXT
}

type browse_context struct {
	Base_url  string
	Name      string
	Parent_id string
	Children  []Object
}

//go:embed templates/browse.tmpl
var browse_tmpl_string string

//go:embed templates/detail.tmpl
var detail_tmpl_string string

//go:embed static
var staticFs embed.FS

func fetchDetail(id int) (*Detail, error) {
	var d Detail
	db, err := sql.Open("sqlite", db_filepath)
	if err != nil {
		return nil, err
	}
	row := db.QueryRow("SELECT PATH, SIZE, TIMESTAMP, TITLE, DURATION, BITRATE, SAMPLERATE, CREATOR, ARTIST, ALBUM, GENRE, COMMENT, CHANNELS,"+
		"DISC, TRACK, DATE, RESOLUTION, THUMBNAIL, ALBUM_ART, ROTATION, DLNA_PN, MIME FROM DETAILS WHERE ID=?", id)
	if err = row.Scan(&d.Path, &d.Size, &d.Timestamp, &d.Title, &d.Duration, &d.Bitrate, &d.Samplerate, &d.Creator, &d.Artist, &d.Album,
		&d.Genre, &d.Comment, &d.Channels, &d.Disc, &d.Track, &d.Date, &d.Resolution, &d.Thumbnail, &d.Album_art, &d.Rotation,
		&d.Dlna_pn, &d.Mime); err != nil {
		fmt.Printf("ERROR scanning id\n")
		return nil, err
	}
	return &d, nil
}

func browseObject(object_id string) (*browse_context, error) {
	var bc browse_context
	db, err := sql.Open("sqlite", db_filepath)
	if err != nil {
		return nil, err
	}
	// fetch the parent_id of the browsed object
	row := db.QueryRow("SELECT PARENT_ID, NAME FROM OBJECTS WHERE OBJECT_ID=?", object_id)
	if err = row.Scan(&bc.Parent_id, &bc.Name); err != nil {
		fmt.Printf("ERROR scanning parent_id\n")
		return nil, err
	}
	// fetch all objects that have the browsed object as parent_id
	rows, err := db.Query("SELECT ID, OBJECT_ID, PARENT_ID, REF_ID, CLASS, DETAIL_ID, NAME FROM OBJECTS WHERE PARENT_ID=?", object_id)
	if err != nil {
		fmt.Printf("ERROR querying all children\n")
		return nil, err
	}
	objects := make([]Object, 0) // todo start with len of rows?
	for rows.Next() {
		var o Object
		if err = rows.Scan(&o.Id, &o.Object_id, &o.Parent_id, &o.Ref_id, &o.Class, &o.Detail_id, &o.Name); err != nil {
			fmt.Printf("ERROR scanning row of children\n")
			return nil, err
		}
		objects = append(objects, o)
	}
	if err = rows.Err(); err != nil {
		fmt.Printf("ERROR gow a rows error\n")
		return nil, err
	}
	bc.Children = objects
	if err = db.Close(); err != nil {
		fmt.Printf("ERROR closing db\n")
		return nil, err
	}
	bc.Base_url = base_url
	return &bc, nil
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	// the ServeMux pattern "/" would catch everything
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/browse", http.StatusSeeOther)
}

func getBrowse(w http.ResponseWriter, r *http.Request) {
	params, _ := url.ParseQuery(r.URL.RawQuery)
	idParam, ok := params["id"]
	if !ok {
		// no id was supplied, default to root
		idParam = []string{"0"}
	}
	// there should be only one id param
	if len(idParam) != 1 {
		io.WriteString(w, "Too many id params")
		return
	}
	bc, err := browseObject(idParam[0])
	if err != nil {
		panic(err) // TODO nfw
	}
	browse_tmpl, err := template.New("browse").Funcs(template.FuncMap{"hasPrefix": strings.HasPrefix}).Parse(browse_tmpl_string)
	if err != nil {
		panic(err)
	}
	browse_tmpl.Execute(w, bc)
}

func getDetail(w http.ResponseWriter, r *http.Request) {
	params, _ := url.ParseQuery(r.URL.RawQuery)
	idParam, ok := params["id"]
	if !ok {
		io.WriteString(w, "Missing id param")
		return
	}
	// there should be only one id param
	if len(idParam) != 1 {
		io.WriteString(w, "Too many id params")
		return
	}
	idint, err := strconv.Atoi(idParam[0])
	if err != nil {
		panic(err) // TODO nfw
	}
	d, err := fetchDetail(idint)
	if err != nil {
		panic(err) // TODO nfw
	}
	detail_tmpl, err := template.New("detail").Parse(detail_tmpl_string)
	if err != nil {
		panic(err)
	}
	detail_tmpl.Execute(w, d)

}

var db_filepath string
var base_url string

func init() {
	flag.StringVar(&db_filepath, "db-file", "", "Path of the minidlna sqlite file, e.g. /var/cache/minidlna/files.db")
	flag.StringVar(&base_url, "base-url", "", "Base URL of the minidlna /MediaItems/ path, e.g. http://hostname:8200/MediaItems/")
}

func main() {
	port := flag.Int("listen-port", 3333, "TCP port on which to listen")
	listenAddr := flag.String("listen-addr", "", "Address on which to listen")
	flag.Parse()

	if base_url == "" {
		fmt.Println("ERROR: Missing base-url cmdline parameter")
		os.Exit(-1)
	}

	if db_filepath == "" {
		fmt.Println("ERROR: Missing db-file cmdline parameter")
		os.Exit(-1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/browse", getBrowse)
	mux.HandleFunc("/detail", getDetail)
	staticFsServer := http.FileServer(http.FS(staticFs))
	mux.Handle("/static/", http.StripPrefix("/", staticFsServer))

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", *listenAddr, *port), mux)
	if err != nil {
		panic(err) // TODO wrong, could be close
	}
}
