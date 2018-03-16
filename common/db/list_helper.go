package db

type ListHelper struct {
	db DB
}

func NewListHelper(db DB) *ListHelper {
	return &ListHelper{db}
}
