package data

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/nhan10132020/greenlight/internal/validator"
	"gorm.io/gorm"
)

type Movie struct {
	ID        int64          `json:"id" gorm:"column:id"`                               // unique interger ID for the movie
	CreatedAt time.Time      `json:"-" gorm:"column:created_at"`                        // Timestamp for when the movie is added to database
	Title     string         `json:"title" gorm:"column:title"`                         // Movie title
	Year      int32          `json:"year,omitempty" gorm:"column:year"`                 // Movie release year
	Runtime   Runtime        `json:"runtime,omitempty" gorm:"column:runtime"`           // Movie runtime(in minutes)
	Genres    pq.StringArray `json:"genres,omitempty" gorm:"column:genres;type:text[]"` // Slice of genres for movie
	Version   int32          `json:"version" gorm:"column:version;default:1"`           // The version number starts at 1 and increment when movie information updated
}

func (Movie) TableName() string { return "movies" }

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive interger")
	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

type MovieModel struct {
	DB *gorm.DB
}

func (m MovieModel) Insert(movie *Movie) error {
	if err := m.DB.Omit("ID", "CreatedAt").Create(movie).Error; err != nil {
		return err
	}
	return nil
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	var movie Movie
	if err := m.DB.Where("id = ?", id).First(&movie).Error; err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &movie, nil
}

func (m MovieModel) Update(movie *Movie) error {
	movie.Version += 1
	if err := m.DB.Save(movie).Error; err != nil {
		return err
	}
	return nil
}

func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	result := m.DB.Delete(&Movie{}, id)

	if err := result.Error; err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}
