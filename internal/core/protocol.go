package core

type Command string

const (
	Status       Command = "status"
	CreateDaily  Command = "create_daily"
	AddWindow    Command = "add_window"
	EditWindow   Command = "edit_window"
	DeleteWindow Command = "delete_window"
	CreateRange  Command = "create_range"
	Confirm      Command = "confirm"
	Cancel       Command = "cancel"
	AddDomain    Command = "add_domain"
	EditDomain   Command = "edit_domain"
	DeleteDomain Command = "delete_domain"
)

type InputWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Request struct {
	Cmd       Command       `json:"cmd"`
	Windows   []InputWindow `json:"windows,omitempty"`
	Start     string        `json:"start,omitempty"`
	End       string        `json:"end,omitempty"`
	Duration  string        `json:"duration,omitempty"`
	Code      string        `json:"code,omitempty"`
	Domain    string        `json:"domain,omitempty"`
	Index     *int          `json:"index,omitempty"`
	Entry     *int          `json:"entry,omitempty"`
	Confirmed bool          `json:"confirmed,omitempty"`
}

type Response struct {
	OK        bool          `json:"ok"`
	Error     ErrorCode     `json:"error,omitempty"`
	Message   string        `json:"message,omitempty"`
	Active    bool          `json:"active"`
	Now       int64         `json:"now"`
	Entries   []EntryStatus `json:"entries,omitempty"`
	Pending   bool          `json:"pending,omitempty"`
	Code      string        `json:"code,omitempty"`
	ExpiresIn int64         `json:"expires_in,omitempty"`
	Failures  int           `json:"failures,omitempty"`
	Preview   *Entry        `json:"preview,omitempty"`
}

type EntryStatus struct {
	Domain  string   `json:"domain"`
	Active  bool     `json:"active"`
	Kind    Kind     `json:"kind,omitempty"`
	Zone    string   `json:"zone,omitempty"`
	EndAt   int64    `json:"end_at,omitempty"`
	NextAt  int64    `json:"next_at,omitempty"`
	Windows []Window `json:"windows,omitempty"`
}
