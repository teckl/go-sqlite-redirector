package main

import (
	"database/sql"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
)

//const redirectHttpStatus = http.StatusFound
const redirectHttpStatus = http.StatusMovedPermanently
const defaultRedirectURLNotExistHostname = "https://default-redirect-url.example.com"

func main() {
	e := echo.New()

	e.Debug = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/*", doRedirect)
	e.HEAD("/*", doRedirect)

	e.Logger.Fatal(e.Start(":1323"))
}

func connectDB(hostname string) (*sql.DB, error) {
	log.Println(fmt.Sprintf("./sqlite/%s.db", hostname))
	return sql.Open("sqlite3", fmt.Sprintf("./sqlite/%s.db", hostname))
}

func doRedirect(c echo.Context) error {
	req := c.Request()

	hostname := req.Host

	scheme := c.Scheme()
	scheme = "https"  // debug only

	host, err := searchHostname(c, scheme, hostname)
	if err != nil {
		c.Logger().Debug("---- searchHostname not found : ", hostname)
		c.Redirect(http.StatusFound, defaultRedirectURLNotExistHostname)
		return nil
	}

	if host == nil || host.isDisabled() {
		c.NoContent(http.StatusNotFound)
		return nil
	}

	p, err := searchPage(c, *host, req.URL.Path, hostname)
	if err != nil {
		c.Error(err)
		return nil
	}

	if p == nil {
		c.Redirect(redirectHttpStatus, host.toHost())
	} else {
		c.Redirect(redirectHttpStatus, fmt.Sprintf("%s%s", host.toHost(), *p))
	}
	return nil
}

type ResHostname struct {
	id     int
	https  bool
	domain string
	status int
}

func (r ResHostname) toHost() string {
	s := "http"
	if r.https {
		s = "https"
	}
	return fmt.Sprintf("%s://%s", s, r.domain)
}

func (r ResHostname) isDisabled() bool {
    return r.status == 0
}

func searchHostname(c echo.Context, scheme, hostname string) (*ResHostname, error) {
	db, err := connectDB(hostname)
	if err != nil {
		c.Logger().Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("SELECT id, to_https, to_domain, status FROM hostname WHERE from_https = ? AND from_domain = ? AND status = 1")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	s := 0
	if scheme == "https" {
		s = 1
	}

	var h ResHostname
	err = stmt.QueryRow(s, hostname).Scan(&h.id, &h.https, &h.domain, &h.status)
	switch {
		case err == sql.ErrNoRows:
			return nil, nil
		case err != nil:
			return nil, err
		default:
			return &h, nil
	}
}

func searchPage(c echo.Context, h ResHostname, path, hostname string) (*string, error) {
	db, err := connectDB(hostname)
	if err != nil {
		c.Logger().Fatal(err)
	}
	defer db.Close()

	stmt, err := db.Prepare("SELECT to_path FROM page WHERE hostname_id = ? AND from_path = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var toPath string

	err = stmt.QueryRow(h.id, path).Scan(&toPath)

	switch {
		case err == sql.ErrNoRows:
			return nil, nil
		case err != nil:
			return nil, err
		default:
			return &toPath, nil
	}
}
