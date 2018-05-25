package config

const (
	DefaultInterval        uint  = 60         // 1 minute
	DefaultMaxSlowLogSize  int64 = 1073741824 // 1G
	DefaultSlowLogRotation       = true       // whether to rotate slow logs
	DefaultRetainSlowLogs        = 1          // how many slow logs to keep on filesystem
	DefaultExampleQueries        = true
	// internal
	DefaultReportLimit uint = 200
)

type QAN struct {
	UUID           string // of MySQL instance
	CollectFrom    string `json:",omitempty"` // "slowlog" or "perfschema"
	Interval       uint   `json:",omitempty"` // seconds, 0 = DEFAULT_INTERVAL
	ExampleQueries *bool  `json:",omitempty"` // send real example of each query
	// "slowlog" specific options.
	MaxSlowLogSize  int64 `json:"-"`          // bytes, 0 = DEFAULT_MAX_SLOW_LOG_SIZE. Don't write it to the config
	SlowLogRotation *bool `json:",omitempty"` // Enable slow logs rotation.
	RetainSlowLogs  *int  `json:",omitempty"` // Number of slow logs to keep.
	// internal
	Start       []string `json:",omitempty"` // queries to configure MySQL (enable slow log, etc.)
	Stop        []string `json:",omitempty"` // queries to un-configure MySQL (disable slow log, etc.)
	ReportLimit uint     `json:",omitempty"` // top N queries, 0 = DEFAULT_REPORT_LIMIT
}

func NewQAN() QAN {
	return QAN{
		Interval:       DefaultInterval,
		ExampleQueries: boolPointer(DefaultExampleQueries),
		// "slowlog" specific options.
		MaxSlowLogSize:  DefaultMaxSlowLogSize,
		SlowLogRotation: boolPointer(DefaultSlowLogRotation),
		RetainSlowLogs:  intPointer(DefaultRetainSlowLogs),
		// internal
		ReportLimit: DefaultReportLimit,
	}
}

// boolValue returns the value of the bool pointer passed in or
// false if the pointer is nil.
func boolPointer(v bool) *bool {
	return &v
}

// boolValue returns the value of the bool pointer passed in or
// false if the pointer is nil.
func intPointer(v int) *int {
	return &v
}
