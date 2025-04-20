package api_types

type StackOverFlowType int

const (
	Answer StackOverFlowType = iota
	Comment
)

func (t StackOverFlowType) String() string {
	switch t {
	case Answer:
		return "answer"
	case Comment:
		return "comment"
	default:
		return ""
	}
}

type StackOverFlowUpdate struct {
	Type  StackOverFlowType
	Title string
	Owner struct {
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	CreatedAt int64  `json:"creation_date"`
	Preview   string `json:"body"`
}
