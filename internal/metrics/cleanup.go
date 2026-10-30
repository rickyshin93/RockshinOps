package metrics

// ScanTarget is a configured cleanup location shown with its scanned size.
type ScanTarget struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
