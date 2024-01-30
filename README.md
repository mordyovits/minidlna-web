![Screenshot of minidlna-web](/minidlna-web%20screenshot.png)

# Overview
minidlna-web provides a web frontend to the SQlite database that minidlna builds for itself. It allows you to use a web browser to explore the media that minidlna is hosting, and it offers downloads links that fetch the media directly from the minildna service.

# Design goals
minidlna-web should be:
1) A single binary file for ease of deployment (including all web assets)
2) Very light and fast (so it can run on the sort of small machines minidlna itself can)
3) Read-only
4) Minimal 3rd party dependencies. Currently only pulls in modernc.org/sqlite
5) Minimally configurable

# Cmdline options
```
$ ./minidlna-web --help
Usage of ./minidlna-web:
  -base-url string
        Base URL of the minidlna /MediaItems/ path, e.g. http://hostname:8200/MediaItems/
  -db-file string
        Path of the minidlna sqlite file, e.g. /var/cache/minidlna/files.db
  -listen-addr string
        Address on which to listen
  -listen-port int
        TCP port on which to listen (default 3333)
```

# Running minidlna
1) Git clone the repo: `git clone https://github.com/mordyovits/minidlna-web.git`
2) In the repo top directory, run `go build`
3) Run the resulting binary with the required cmdline args, e.g.: `./minidlna-web -base-url http://192.168.1.XXX:8200/MediaItems/ -db-file /var/cache/minidlna/files.db`

**Note:** If you want minidlna-web to read the main sqlite database file that minidlna built and is maintaining then you will probably need to run minidlna as the minidlna role user account (usually `minidlna`).

# TODO
* Add search functionality
* Deal with caching headers (version the static assets?)
* Use template inheritance to DRY
* Add FastCGI support so it can run under a web server (which can provide TLS, authentication, and authorization)
* Improve the HTML and web design, which is very crude
* Add systemd unit file
* Add RPM build (spec) and DPKG build files

# Contributing
I would most appreciate help with the web design aspects. The project is set up to be pretty easy to contribute that work to: the HTML is in templates/ and the static assets are in static/.



