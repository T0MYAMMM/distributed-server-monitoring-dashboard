package domain

// Log levels follow the log-geulis standard (TS | LEVEL | MODULE | MESSAGE).
const (
	LogDebug = "DEBUG"
	LogInfo  = "INFO"
	LogWarn  = "WARN"
	LogError = "ERROR"
)

// LogLine is one structured log entry shipped by an agent and stored in the
// external log database. Ts is ISO-8601 (UTC) on the wire.
type LogLine struct {
	ID         int64  `json:"id"`
	Server     string `json:"server"`
	Ts         string `json:"ts"`
	Level      string `json:"level"`
	Module     string `json:"module"`
	Message    string `json:"message"`
	SourceFile string `json:"source_file"`
}

// LogQuery filters a log search. Empty fields are ignored. AfterID > 0 selects
// only newer rows (used by the live-tail stream). Search is a keyword match on
// the message only; Module filters by app/module exactly.
type LogQuery struct {
	ServerID string
	Level    string
	Modules  []string
	Search   string
	Since    string
	Until    string
	File     string
	Limit    int
	AfterID  int64
}
