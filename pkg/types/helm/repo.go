package helm

type Repo struct {
	Name   string `json:"name"`
	Url    string `json:"url"`
	Source string `json:"source"`
}
