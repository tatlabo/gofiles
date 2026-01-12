package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func MakeTestDirectories() Directries {

	id, _ := uuid.Parse("d5615d5c-be44-4913-965c-593ca75dde60")
	createdAt, _ := time.Parse(time.RFC3339, "2026-01-09 14:14:07.070389 +0000 GMT")
	updatedAt, _ := time.Parse(time.RFC3339, "2026-01-09 14:16:55.473366 +0000 GMT")

	d := Directory{
		Id:        id,
		Path:      `Z:\AFLOFARM_ONILNE\250113_Aflofarm_Proliver_diabeto`,
		IsDone:    true,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	var t Directries
	t.Array = append(t.Array, d)

	return t
}

func TestList(t *testing.T) {

	testDirs := MakeTestDirectories()

	d := Directries{}

	err := d.List()
	if err != nil {
		t.Errorf("Directries.List() error = %v", err)
		return
	}

	if d.Array[0].Path != testDirs.Array[0].Path {
		t.Errorf("Directries.List() = %v, want %v", d.Array[0].Path, testDirs.Array[0].Path)
	}

}
