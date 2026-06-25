package domain

// Contact is one way to reach the owner; an empty URL renders Value as plain text.
type Contact struct {
	Label string
	Value string
	URL   string
}
