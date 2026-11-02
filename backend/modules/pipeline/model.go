package pipeline

type PipelineResult struct {
	TotalScanned int `json:"total_scanned"`
	NewDrafts    int `json:"new_drafts"`
	Skipped      int `json:"skipped"`
	Errors       int `json:"errors"`
}
