// Package models
// Direcory
// Directories
package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Directory struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Path      string    `json:"path" db:"path"`
	IsDone    bool      `json:"isDone" db:"is_done"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type DirectoryModel struct {
	DB *pgxpool.Pool
}

type Directries struct {
	Array []Directory       `json:"list"`
	Body  map[string]string `json:"body"`
	Title string            `json:"title"`
}

type Dir interface {
	AddPath(path string) error
}

func (ds *DirectoryModel) AddPath(path string) error {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `INSERT INTO directory (path, is_done, created_at, updated_at) VALUES ($1, false, NOW(), NOW());`

	d := Directory{}
	err := ds.DB.QueryRow(ctx, query, path).Scan(&d.ID)
	if err != nil {
		return err
	}

	return nil
}

func (ds *DirectoryModel) DeletePath(id uuid.UUID) (ID int, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `DELETE FROM directory WHERE id = $1 RETURNING id`

	err = ds.DB.QueryRow(ctx, query, id).Scan(&ID)
	if err != nil {
		return 0, err
	}

	return ID, nil
}

func (ds *DirectoryModel) Direcotry(id uuid.UUID) (d Directory, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory WHERE id = $1;`

	err = ds.DB.QueryRow(ctx, query, id).Scan(&d.ID, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}

	return d, nil
}

func (ds *DirectoryModel) List() ([]Directory, error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory;`

	rows, err := ds.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	list := []Directory{}

	for rows.Next() {
		d := Directory{}

		err := rows.Scan(&d.ID, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}

		list = append(list, d)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (ds *DirectoryModel) Row(id uuid.UUID) (d Directory, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory WHERE id = $1;`

	err = ds.DB.QueryRow(ctx, query, id).Scan(&d.ID, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}

	return d, nil
}
