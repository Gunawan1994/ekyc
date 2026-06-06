package pagination

import (
	"math"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

// ParseParams extracts pagination and filter parameters from the request
// query string. Missing or invalid values fall back to safe defaults.
func ParseParams(c echo.Context) domain.ListParams {
	page := parseInt(c.QueryParam("page"), defaultPage)
	if page < 1 {
		page = defaultPage
	}

	pageSize := parseInt(c.QueryParam("page_size"), defaultPageSize)
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	sortDir := c.QueryParam("sort_dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	return domain.ListParams{
		Page:     page,
		PageSize: pageSize,
		Search:   c.QueryParam("search"),
		Status:   c.QueryParam("status"),
		SortBy:   c.QueryParam("sort_by"),
		SortDir:  sortDir,
	}
}

// CalcPages returns the total number of pages given a record count and page
// size. Returns 0 when pageSize is zero to avoid division by zero.
func CalcPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

// Offset computes the SQL OFFSET value for the given page and page size.
// Page is 1-indexed; values below 1 are treated as 1.
func Offset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}

// parseInt converts a query-param string to int, returning fallback on
// empty or invalid input.
func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
