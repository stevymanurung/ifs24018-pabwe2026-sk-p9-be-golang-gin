package post

import "testing"

func TestSlugify(t *testing.T) {
	got := Slugify("  Belajar Go & Gin!  ")
	if got != "belajar-go-gin" {
		t.Fatalf("got %q", got)
	}
	if Slugify("---") != "" {
		t.Fatal("expected empty slug")
	}
}
func TestReadingTime(t *testing.T) {
	if ReadingTime("") != 1 {
		t.Fatal("empty content must be at least one minute")
	}
	if ReadingTime("one two three") != 1 {
		t.Fatal("short content")
	}
	words := ""
	for i := 0; i < 201; i++ {
		words += "kata "
	}
	if ReadingTime(words) != 2 {
		t.Fatal("201 words should take two minutes")
	}
}
