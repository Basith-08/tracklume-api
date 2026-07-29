package response

import "testing"

func TestNormalizePagination(t *testing.T) {
	if page, perPage := NormalizePagination(0, 1000); page != 1 || perPage != 100 {
		t.Fatalf("got %d/%d", page, perPage)
	}
	if meta := PaginationMeta(1, 20, 41); meta.TotalPages != 3 {
		t.Fatalf("got %d pages", meta.TotalPages)
	}
}
