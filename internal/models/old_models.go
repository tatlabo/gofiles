package models

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

func (s *SearchParams) QueryStmt() error {

	switcher := 0
	isOn := []string{"on", "true", "1", "yes", "ok", "y", "tak", "t"}

	if slices.Contains(isOn, s.Like) {
		switcher += 1
	}

	if slices.Contains(isOn, s.Dir) {
		switcher += 100
	}

	language := "'polish'"
	if len(s.Keywords) > 0 {
		language = "'english'"
	}

	s.QueryParam = s.Params

	if strings.Contains(s.Params, ".") {
		// Regex: split on dot, comma, semicolon, or any whitespace
		re := regexp.MustCompile(`[.,;]+`)
		parts := re.Split(s.Params, -1)
		s.QueryParam = parts[0]
		s.QueryParam = strings.TrimSpace(s.QueryParam)
		s.Ext = parts[1]
	}

	switch switcher {
	case 0:

		// column = keywords
		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, directory, size, mod_time,
		ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
		FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
		ORDER BY ts_rank DESC
		LIMIT $2 OFFSET $3;`, language)

		s.CounterStmt = fmt.Sprintf(`
		SELECT COUNT(id) FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, language)

	case 1:
		s.QueryParam = "^" + s.QueryParam
		// column = "name"
		s.Stmt = `
		SELECT id, name, ext, is_dir, directory, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1)
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`

		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1);`
	case 100:

		// column = "files.keywords"

		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, directory, size, mod_time FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, language)

		s.CounterStmt = fmt.Sprintf(`
		SELECT COUNT(*) FROM files 
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND is_dir=true;`, language)

	case 101:
		s.QueryParam = "^" + s.QueryParam

		s.Stmt = `
		SELECT id, name, ext, is_dir, directory, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1) AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`
		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1) AND is_dir=true;`
	}

	s.Placeholders = []any{s.QueryParam, s.Limit, s.Offset}

	if len(s.Ext) > 0 {

		s.Stmt = fmt.Sprintf(`SELECT files.id, name, files.ext, is_dir, directory, size, mod_time,
		ts_rank( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
		FROM files
		JOIN ext ON files.ext_id = ext.id
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND ext.ext = $2
		ORDER BY ts_rank DESC LIMIT $3 OFFSET $4;`, language)

		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files
		JOIN ext ON files.ext_id = ext.id
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND ext.ext = $2;`, language)
		s.Placeholders = []any{s.Placeholders[0], s.Ext, s.Placeholders[1], s.Placeholders[2]}
	}

	return nil

}
