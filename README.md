# go-sqlite-redirector

MEMO

Prepare a SQLite file for each target domain to handle huge numbers of redirections.

```
$ sqlite3 hoge.example.com.db < sql.sql
$ sqlite3 hoge.example.com.db < hoge.sql

$ sqlite3 foo.example.com.db < sql.sql
$ sqlite3 foo.example.com.db < foo.sql

$ sqlite3 bar.example.com.db < sql.sql
$ sqlite3 bar.example.com.db < bar.sql
```
