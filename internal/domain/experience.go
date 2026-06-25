package domain

// Experience is one position. Start and End are ISO YYYY-MM sort keys only and
// are never rendered; an empty End means present.
type Experience struct {
	Company     string
	Role        string
	Period      string
	Location    string
	Arrangement string // remote / hybrid / on-site / contract
	Summary     string
	Highlights  []string
	Stack       []string
	Current     bool
	Start       string
	End         string
}
