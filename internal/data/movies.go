package data

import "time"

type Movie struct {
	ID        int64     `json:"id"`                // unique interger ID for the movie
	CreatedAt time.Time `json:"-"`                 // Timestamp for when the movie is added to database
	Title     string    `json:"title"`             // Movie title
	Year      int32     `json:"year,omitempty"`    // Movie release year
	Runtime   Runtime   `json:"runtime,omitempty"` // Movie runtime(in minutes)
	Genres    []string  `json:"genres,omitempty"`  // Slice of genres for movie
	Version   int32     `json:"version"`           // The version number starts at 1 and increment when movie information updated
}
