package models

import (
	"gofiles/utils"
)

func (d *Directory) AddPath(path string) error {
	const query = `INSERT INTO directory (path, is_done, created_at, updated_at) VALUES ($1, false, NOW(), NOW()) RETURNING id, path, is_done, created_at, updated_at;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}

	err = conn.QueryRow(query, path).Scan(&d.ID, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}
