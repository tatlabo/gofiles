// Package models
// Direcory
// Directories
package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type Direr interface {
	AddPath(path string) error
	Row(UUID uuid.UUID) (d Directory, err error)
	Rows() ([]Directory, error)
	Delete(UUID uuid.UUID) error
}

func (ds *DirectoryModel) AddPath(path string) error {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `INSERT INTO directory (path, is_done, created_at, updated_at) VALUES ($1, false, NOW(), NOW());`

	comm, err := ds.DB.Exec(ctx, query, path)
	if err != nil {
		return err
	}
	if !comm.Insert() {
		return fmt.Errorf("insert failed")
	}

	return nil
}

func (ds *DirectoryModel) Row(UUID uuid.UUID) (d Directory, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `SELECT id, path, is_done, created_at, updated_at FROM directory WHERE id = @UUID;`

	args := pgx.NamedArgs{
		"UUID": UUID,
	}

	err = ds.DB.QueryRow(ctx, query, args).Scan(&d.ID, &d.Path, &d.IsDone, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}

	return d, nil
}

func (ds *DirectoryModel) Rows() ([]Directory, error) {
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

func (ds *DirectoryModel) Delete(UUID uuid.UUID) error {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const query = `DELETE FROM directory WHERE id = @UUID`

	args := pgx.NamedArgs{
		"UUID": UUID,
	}

	comm, err := ds.DB.Exec(ctx, query, args)
	if err != nil {
		return err
	}
	if !comm.Delete() {
		return fmt.Errorf("insert failed")
	}

	return nil
}
