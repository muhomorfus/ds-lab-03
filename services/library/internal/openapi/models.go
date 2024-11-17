package openapi

import "github.com/google/uuid"

type library struct {
	ID         int       `db:"id"`
	LibraryUID uuid.UUID `db:"library_uid"`
	Name       string    `db:"name"`
	City       string    `db:"city"`
	Address    string    `db:"address"`
}

type book struct {
	ID        int       `db:"id"`
	BookUID   uuid.UUID `db:"book_uid"`
	Name      string    `db:"name"`
	Author    string    `db:"author"`
	Genre     string    `db:"genre"`
	Condition string    `db:"condition"`
}

type libraryBook struct {
	ID             int       `db:"id"`
	BookUID        uuid.UUID `db:"book_uid"`
	AvailableCount int       `db:"available_count"`
	Name           string    `db:"name"`
	Author         string    `db:"author"`
	Genre          string    `db:"genre"`
	Condition      string    `db:"condition"`
}

type libraryBookRaw struct {
	LibraryID      int `db:"library_id"`
	BookID         int `db:"book_id"`
	AvailableCount int `db:"available_count"`
}
