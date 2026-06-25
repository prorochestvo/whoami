package domain

type Resume struct {
	Person     Person
	About      []string
	Projects   []Project
	Experience []Experience
	Education  []Education
	Skills     []SkillGroup
	Languages  []Language
	Relocation []string
	Contacts   []Contact
}
