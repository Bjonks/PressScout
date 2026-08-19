package model

type Classification string

const (
	OK           Classification = "OK"
	Redirect     Classification = "REDIRECT"
	Broken       Classification = "BROKEN"
	Forbidden    Classification = "FORBIDDEN"
	RateLimited  Classification = "RATE_LIMITED"
	ServerError  Classification = "SERVER_ERROR"
	Timeout      Classification = "TIMEOUT"
	NetworkError Classification = "NETWORK_ERROR"
)

type Result struct {
	OriginalURL string         `json:"original_url"`
	FinalURL    string         `json:"final_url,omitempty"`
	Status      int            `json:"status,omitempty"`
	Error       string         `json:"error,omitempty"`
	External    bool           `json:"external"`
	Sources     []string       `json:"sources"`
	Class       Classification `json:"classification"`
}
