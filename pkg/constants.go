package pkg

type (
	AvailableHosts   = string
	TransportType    = string
	StateManagerType = string
	SiteAddress      = string
)

const (
	MaxPreviewLen                         = 200
	Stackoverflow        AvailableHosts   = "stackoverflow.com"
	Github               AvailableHosts   = "github.com"
	TransportTypeHTTP    TransportType    = "http"
	TransportTypeKafka   TransportType    = "kafka"
	RedisStateManager    StateManagerType = "Redis"
	InMemoryStateManager StateManagerType = "In-memory"
	StackOverflowAddress SiteAddress      = "https://api.stackexchange.com/2.3"
	GithubAddress        SiteAddress      = "https://api.github.com"
)
