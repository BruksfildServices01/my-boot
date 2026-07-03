package models

import "time"

type MediaType string

const (
	MediaTypeVideo MediaType = "video"
	MediaTypeImage MediaType = "image"
)

type Testimonial struct {
	ID        string
	Type      MediaType
	Caption   string
	URL       string    // YouTube URL (video) ou R2 URL (image)
	Visible   bool
	SortOrder int
	CreatedAt time.Time
}

// YouTubeID extrai o ID do vídeo de uma URL do YouTube.
// Suporta: youtube.com/watch?v=ID, youtu.be/ID, youtube.com/shorts/ID
func (t *Testimonial) YouTubeID() string {
	if t.Type != MediaTypeVideo {
		return ""
	}
	u := t.URL
	// youtu.be/ID
	if idx := indexOf(u, "youtu.be/"); idx >= 0 {
		id := u[idx+9:]
		if i := indexOfAny(id, "?&"); i >= 0 {
			id = id[:i]
		}
		return id
	}
	// youtube.com/shorts/ID
	if idx := indexOf(u, "/shorts/"); idx >= 0 {
		id := u[idx+8:]
		if i := indexOfAny(id, "?&"); i >= 0 {
			id = id[:i]
		}
		return id
	}
	// youtube.com/watch?v=ID
	if idx := indexOf(u, "v="); idx >= 0 {
		id := u[idx+2:]
		if i := indexOfAny(id, "&"); i >= 0 {
			id = id[:i]
		}
		return id
	}
	return ""
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func indexOfAny(s, chars string) int {
	for i, c := range s {
		for _, ch := range chars {
			if c == ch {
				return i
			}
		}
	}
	return -1
}
