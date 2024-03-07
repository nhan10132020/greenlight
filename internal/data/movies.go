package data

import (
	"context"
	"errors"
	"fmt"
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
	Version   int32          `json:"version" gorm:"column:version"`                     // The version number starts at 1 and increment when movie information updated
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
	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.DB.WithContext(ctx).Omit("ID", "CreatedAt", "Version").Create(movie).Error; err != nil {
		return err
	}
	movie.Version = 1
	return nil
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	var movie Movie

	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.DB.WithContext(ctx).Where("id = ?", id).First(&movie).Error; err != nil {
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

	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// at condition on "version" field to avoid data race existing
	result := m.DB.
		WithContext(ctx).
		Model(&movie).
		Where("version = ?", movie.Version-1).
		Omit("ID", "CreatedAt").
		Updates(movie)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrEditConflict
	}
	return nil
}

func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := m.DB.WithContext(ctx).Delete(&Movie{}, id)

	if err := result.Error; err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	// Example Query for this method :
	// SELECT count(*) OVER() ,id, created_at, title, year, runtime, genres, version
	// FROM movies
	// WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', ?) OR ? = '') AND (genres @> ? OR ? = '{}')
	// ORDER BY filters.sortColumn() filters.sortDirection(), id ASC
	// LIMIT filters.limit() OFFSET filters.offset()

	// context 3-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	movies := []*Movie{}
	var totalRecords int64

	if err := m.DB.
		WithContext(ctx).
		Table("movies").
		Where("(to_tsvector('simple', title) @@ plainto_tsquery('simple', ?) OR ? = '') AND (genres @> ? OR ? = '{}')", title, title, pq.StringArray(genres), pq.StringArray(genres)).
		Count(&totalRecords).
		Order(fmt.Sprintf("%s %s, id ASC", filters.sortColumn(), filters.sortDirection())).
		Limit(filters.limit()).
		Offset(filters.offset()).
		Scan(&movies).
		Error; err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(int(totalRecords), filters.Page, filters.PageSize)

	return movies, metadata, nil
}
