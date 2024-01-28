package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"strings"
	// "os"
	// "time"
	"html/template"
	_ "modernc.org/sqlite"
	"net/http"
	"path/filepath"
)

type Object struct {
	Id int64 //   ID INTEGER PRIMARY KEY AUTOINCREMENT
	Object_id string // OBJECT_ID TEXT UNIQUE NOT NULL
	Parent_id string // PARENT_ID TEXT NOT NULL
	Ref_id sql.NullString // REF_ID TEXT DEFAULT NULL
	Class string // CLASS TEXT NOT NULL
	Detail_id int64 // DETAIL_ID INTEGER DEFAULT NULL
	Name string // NAME TEXT DEFAULT NULL
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
	Parent_id string
	Children []Object
}


var db_filename = "files2.db"

//db_directory := "/var/cache/minidlna"
var db_directory = "."
var db_fullpath = filepath.Join(db_directory, db_filename)

var root_tmpl_string = "<html><head><title>Root</title></head><body>All details:<hr/>" +
					   "<table><tr><th>ID</th><th>PATH</th><th>SIZE</th><th>TIMESTAMP</th><th>TITLE</th><th>DURATION</th><th>BITRATE</th></tr>" +
                       "{{ range . }}<tr>" +
					   "<td><a href=\"http://irac:8200/MediaItems/{{ .Id }}\">{{ .Id }}</a><td>" +
					   "{{ .Path }}</td>" +
					   "<td>{{ .Size }}</td>" +
					   "<td>{{ .Timestamp }}</td>" +
					   "<td>{{ .Title }}</td>" +
					   "<td>{{ .Duration }}</td>" +
					   "<td>{{ .Bitrate }}</td>" +
					   "</tr>{{ end }}" +
					   "</table></body></html>"

var browse_tmpl_string = "<html><head><title>Browse Object</title></head><body>" +
                         "Parent: <a href=\"/browse?id={{ .Parent_id }}\">UP</a><hr/>" +
						 "<ul>" +
						 "{{ range .Children }}" +
						 "{{if hasPrefix .Class \"container\"}}" +
						 "<li><a href=\"/browse?id={{ .Object_id }}\">{{ .Name }}</a></li>" +
						 "{{ else }}" +
						 "<li><a href=\"http://192.168.1.193:8200/MediaItems/{{ .Detail_id }}-{{ .Name }}\">{{ .Name }}</a></li>" +
						 "{{ end }}" +
						 "{{ end }}</ul>" +						 
						 "</body></html>"



func fetchAllDetails() ([]Detail, error) {
	details := make([]Detail, 0)
	db, err := sql.Open("sqlite", db_fullpath)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT id, path, size, timestamp, title, duration, bitrate, samplerate, " +
		"creator, artist, album, genre, comment, channels, disc, track, date, " +
		"resolution, thumbnail, album_art, rotation, dlna_pn, mime FROM DETAILS;")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d Detail
		// fmt.Printf("%+v\n", rows)
		if err = rows.Scan(&d.Id, &d.Path, &d.Size, &d.Timestamp, &d.Title, &d.Duration, &d.Bitrate,
			&d.Samplerate, &d.Creator, &d.Artist, &d.Album, &d.Genre, &d.Comment,
			&d.Channels, &d.Disc, &d.Track, &d.Date, &d.Resolution, &d.Thumbnail,
			&d.Album_art, &d.Rotation, &d.Dlna_pn, &d.Mime); err != nil {
			return nil, err
		}
		details = append(details, d)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if err = db.Close(); err != nil {
		return nil, err
	}
	return details, nil
}

/*
func fetchDetail(id int) (Detail, error) {
	db, err := sql.Open("sqlite", db_fullpath)
	if err != nil {
		return nil, err
	}
}
*/

func browseObject(object_id string) (*browse_context, error ) {
	var bc browse_context
	db, err := sql.Open("sqlite", db_fullpath)
	if err != nil {
		return nil, err
	}
	//var p_id string
	// fetch the parent_id of the browsed object
	row := db.QueryRow("SELECT PARENT_ID FROM OBJECTS WHERE OBJECT_ID=?", object_id)
	if err = row.Scan(&bc.Parent_id); err != nil {
		fmt.Printf("ERROR scanning parent_id\n")
		return nil, err
	}
	// bc.Parent_id = p_id
	fmt.Printf("%s\n", bc)
	fmt.Printf("got objects_id= %s and found parent_id = %s\n", object_id, bc.Parent_id)
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
			fmt.Printf("ERROR scanning row of schildren\n")
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

	return &bc, nil
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	details, err := fetchAllDetails()
	if err != nil {
		panic(err)
	}
	root_tmpl, err := template.New("root").Parse(root_tmpl_string)
	if err != nil {
		panic(err)
	}
	root_tmpl.Execute(w, details)
	//io.WriteString(w, "This is my website!\n")
}

func getBrowse(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /browse request\n")
	fmt.Printf("url: %v\n", r.URL)
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
	fmt.Printf("url: %v\n", idParam)
    //io.WriteString(w, "Got browse")
	bc, err := browseObject(idParam[0])
	if err != nil {
		panic(err) // TODO nfw
	}
	fmt.Printf("bc = %v\n", bc)
	//fmt.Printf("browse, parent_id = %s\n", parent_id)
	//fmt.Printf("browse, children = %v\n", children)
	browse_tmpl, err := template.New("browse").Funcs(template.FuncMap{"hasPrefix": strings.HasPrefix,}).Parse(browse_tmpl_string)
	if err != nil {
		panic(err)
	}
	fmt.Printf("browse_tmpl = %s\n", browse_tmpl)
	browse_tmpl.Execute(w, bc)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	mux.HandleFunc("/browse", getBrowse)
	err := http.ListenAndServe(":3333", mux)
	if err != nil {
		panic(err) // wrong, could be close
	}
}
