package db

type Profile struct {
	Name       string
	Handle     string
	Location   string
	DOB        string
	Tagline    string
	HeroBio    string
	AboutPara1 string
	AboutPara2 string
	Avatar     string
}

type SocialLink struct {
	ID       int64
	Platform string
	URL      string
	Label    string
	Sort     int
}

type Project struct {
	ID          int64
	Name        string
	URL         string
	Description string
	Icon        string
	IsLink      bool
	URLLabel    string
	Sort        int
}

type Experience struct {
	ID      int64
	Years   string
	Role    string
	Company string
	Icon    string
	Sort    int
}

type Skill struct {
	ID   int64
	Name string
	Sort int
}

type Theme struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	TokensBase  map[string]string
	TokensLight map[string]string
	TokensDark  map[string]string
}

type SiteContent struct {
	Profile    Profile
	Social     []SocialLink
	Projects   []Project
	Experience []Experience
	Skills     []Skill
	Theme      Theme
	Settings   map[string]string
}
