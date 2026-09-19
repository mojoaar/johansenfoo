package db

type Profile struct {
	Name       string `json:"name"`
	Handle     string `json:"handle"`
	Location   string `json:"location"`
	DOB        string `json:"dob"`
	Tagline    string `json:"tagline"`
	HeroBio    string `json:"hero_bio"`
	Bio        string `json:"bio"`
	AboutPara1 string `json:"about_para_1"`
	AboutPara2 string `json:"about_para_2"`
	Avatar     string `json:"avatar"`
}

type SocialLink struct {
	ID       int64  `json:"id"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Label    string `json:"label"`
	Sort     int    `json:"sort"`
	Visible  bool   `json:"visible"`
}

type Project struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	IsLink      bool   `json:"is_link"`
	URLLabel    string `json:"url_label"`
	Sort        int    `json:"sort"`
	Visible     bool   `json:"visible"`
}

type Experience struct {
	ID      int64  `json:"id"`
	Years   string `json:"years"`
	Role    string `json:"role"`
	Company string `json:"company"`
	Icon    string `json:"icon"`
	Sort    int    `json:"sort"`
	Visible bool   `json:"visible"`
}

type Skill struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Sort    int    `json:"sort"`
	Visible bool   `json:"visible"`
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
