package apitypes

type GithubType int

const (
	PR GithubType = iota
	Issue
)

func (t GithubType) String() string {
	switch t {
	case PR:
		return "Pull Request"
	case Issue:
		return "Issue"
	default:
		return ""
	}
}

type GithubUpdate struct {
	Type   GithubType
	Title  string `json:"title"`
	Author struct {
		Name string `json:"login"`
	} `json:"user"`
	LastUpdateNumber int    `json:"number"`
	CreatedAt        string `json:"created_at"`
	Preview          string `json:"body"`
}
