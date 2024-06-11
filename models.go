package main
type ClassificationLogModel struct {
	Id                 string                   `json:"id"`
	Timestamp          string                   `json:"timestamp"`
	TenantId           int                      `json:"tenantId"`
	JobID              int                      `json:"jobId"`
	Asset              string                   `json:"asset"`
	ParentAsset        string                   `json:"parentAsset"`
	SourceType         string                   `json:"sourceType"`
	RootAsset          string                   `json:"rootAsset"`
	ClassificationType string                   `json:"classificationType"`
	InfoType           string                   `json:"infoType"`
	FileIdentifiers    []string                 `json:"fileIdentifiers"` //SEND COMMA SEPARATED IDENTIFIERS
	Identifiers        []map[string]interface{} `json:"identifiers"`
	FileSizeInBytes    int64                    `json:"fileSize"`
	AgentID            int64                    `json:"agentId"`
	BlockNum           int64                    `json:"blockNum"`
	LastAccessedAt     string                   `json:"lastAccessedAt"`
	LastModifiedAt     string                   `json:"lastModifiedAt"`
	Labels             []map[string]interface{} `json:"labels"`
	RunId              int                      `json:"runId"`
}
