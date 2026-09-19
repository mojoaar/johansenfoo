package db

import "time"

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

type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Post struct {
	ID             int64      `json:"id"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	BodyMD         string     `json:"body_md"`
	Status         string     `json:"status"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	HeroImageURL   string     `json:"hero_image_url"`
	HeroImageAlt   string     `json:"hero_image_alt"`
	SEOTitle       string     `json:"seo_title"`
	SEODescription string     `json:"seo_description"`
	OGImageURL     string     `json:"og_image_url"`
	CanonicalURL   string     `json:"canonical_url"`
	NoIndex        bool       `json:"noindex"`
	Tags           []Tag      `json:"tags"`
}

type PageSeo struct {
	ID           int64  `json:"id"`
	Route        string `json:"route"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	OGImageURL   string `json:"og_image_url"`
	CanonicalURL string `json:"canonical_url"`
	NoIndex      bool   `json:"noindex"`
}

type Theme struct {
	ID          int64             `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	TokensBase  map[string]string `json:"tokens_base"`
	TokensLight map[string]string `json:"tokens_light"`
	TokensDark  map[string]string `json:"tokens_dark"`
	Sort        int               `json:"sort"`
}

type SiteContent struct {
	Profile    Profile
	Social     []SocialLink
	Projects   []Project
	Experience []Experience
	Skills     []Skill
	Theme      Theme
	PageSeo    map[string]PageSeo
	Settings   map[string]string
}
