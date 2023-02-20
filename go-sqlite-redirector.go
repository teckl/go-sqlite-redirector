package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	db, err := sql.Open("sqlite3", "./main.db")
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()

	h := NewHandler(db)
	e.GET("/*", h.Page)

	e.Logger.Fatal(e.Start(":8080"))
}

type handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *handler {
	return &handler{db}
}

func (h handler) Page(c echo.Context) error {
	req := c.Request()
	host, err := searchHostname(h.db, c.Scheme(), req.Host)
	if err != nil {
		c.Error(err)
		return nil
	}

	// 該当がない or 無効なので
	if host == nil || host.isDisabled() {
		c.NoContent(http.StatusNotFound)
		return nil
	}

	// トップのみリダイレクト
	if host.isTopPageOnly() {
		c.Redirect(http.StatusFound, host.toHost())
		return nil
	}

	p, err := searchPage(h.db, *host, req.URL.Path)
	if err != nil {
		c.Error(err)
		return nil
	}

	if p == nil {
		c.Redirect(http.StatusFound, host.toHost())
	} else {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s%s", host.toHost(), *p))
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

func (r ResHostname) isTopPageOnly() bool {
    return r.status == 2
}

func searchHostname(db *sql.DB, scheme, hostname string) (*ResHostname, error) {
	stmt, err := db.Prepare("select hostname_id, to_https, to_domain, status from hostname where from_https = ? AND from_domain = ?")
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

func searchPage(db *sql.DB, h ResHostname, path string) (*string, error) {
	stmt, err := db.Prepare("select to_path from page where hostname_id = ? AND from_path = ?")
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
