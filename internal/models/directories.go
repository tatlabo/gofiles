package models

import (
	"fmt"
	"gofiles/utils"
	"log"
	"time"

	"github.com/google/uuid"
)

type Directory struct {
	Id        uuid.UUID `json:"id" db:"id"`
	Path      string    `json:"path" db:"path"`
	IsDone    bool      `json:"isDone" db:"is_done"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Directries struct {
	Array []Directory       `json:"list"`
	Body  map[string]string `json:"body"`
	Title string            `json:"title"`
}

type Dir interface {
	AddPath(path string) error
}

func (l *Directries) AddPath(path string) error {
	const query = `INSERT INTO directory (path, is_done, created_at, updated_at) VALUES ($1, false, NOW(), NOW()) RETURNING id, path, is_done, created_at, updated_at;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	d := Directory{}
	err = conn.QueryRow(query, path).Scan(&d.Id, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (d *Directory) AddPath(path string) error {
	const query = `INSERT INTO directory (path, is_done, created_at, updated_at) VALUES ($1, false, NOW(), NOW()) RETURNING id, path, is_done, created_at, updated_at;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.QueryRow(query, path).Scan(&d.Id, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (l *Directries) DeletePath(id uuid.UUID) (d Directory, err error) {
	const query = `DELETE FROM directory WHERE id = $1 RETURNING id, path, is_done, created_at, updated_at;`

	conn, err := utils.PgConn()
	if err != nil {
		return d, err
	}
	defer conn.Close()

	err = conn.QueryRow(query, id).Scan(&d.Id, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}

	return d, nil
}

func (ds *Directries) Direcotry(id uuid.UUID) (d Directory, err error) {
	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory WHERE id = $1;`
	conn, err := utils.PgConn()
	if err != nil {
		return d, err
	}
	defer conn.Close()

	err = conn.QueryRow(query, id).Scan(&d.Id, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}

	return d, nil
}

func (l *Directries) List() error {
	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(query)
	if err != nil {
		return err
	}

	for rows.Next() {

		d := Directory{}

		err := rows.Scan(&d.Id, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return err
		}

		l.Array = append(l.Array, d)
		log.Println("Directory:", d)
	}

	return nil
}

func (d *Directory) Row(id uuid.UUID) error {
	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory WHERE id = $1;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.QueryRow(query, id).Scan(&d.Id, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)

	if err != nil {
		return err
	}

	return nil
}

type IndexedDir struct {
	Id      uuid.UUID `db:"id" json:"id"`
	Name    string    `db:"name" json:"name"`
	Done    bool      `db:"done" json:"done"`
	Created time.Time `db:"created" json:"created"`
	Updated time.Time `db:"updated" json:"updated"`
}

type IndexedDirs struct {
	Indexeddirs []IndexedDir `json:"indexedDirs"`
	Text        string
	HeaderTitle string
	Status      bool
	Params      map[string]string
	Error       map[string]string
}

type User struct {
	Id       uuid.UUID `db:"id" json:"id"`
	Username string    `db:"username" json:"username"`
	Password string    `db:"password" json:"-"`
}

func (i *IndexedDirs) List() error {

	const query = `SELECT id, name, done, created FROM directory ORDER BY created DESC;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query indexed directories: %w", err)
	}

	for rows.Next() {
		var dir IndexedDir
		if err := rows.Scan(&dir.Id, &dir.Name, &dir.Done, &dir.Created); err != nil {
			return fmt.Errorf("failed to scan indexed directory: %w", err)
		}

		i.Indexeddirs = append(i.Indexeddirs, dir)
	}

	if len(i.Indexeddirs) == 0 {
		i.Text = "No indexed directories found."
	}

	return nil
}

func (i *IndexedDirs) Append() error {

	query := `INSERT INTO directory (name, done, created) VALUES ($1, $2, $3) RETURNING id, name, done, created;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	newDir := IndexedDir{}
	err = conn.QueryRow(query, i.Params["path"], false, time.Now()).Scan(
		&newDir.Id, &newDir.Name, &newDir.Done, &newDir.Created)
	if err != nil {
		return fmt.Errorf("failed to insert into indexed directories: %w", err)
	}

	// Add the new directory to the slice
	i.Indexeddirs = append(i.Indexeddirs, newDir)

	return nil

}
