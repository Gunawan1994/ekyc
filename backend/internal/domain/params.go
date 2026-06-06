package domain

type ListParams struct {
	Page     int
	PageSize int
	Search   string
	Status   string
	SortBy   string
	SortDir  string
}
