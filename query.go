package spisql

type Query struct {
	Filters    Filters
	Sorts      Sorts
	Pagination Pagination
	Groups     Groups
}
