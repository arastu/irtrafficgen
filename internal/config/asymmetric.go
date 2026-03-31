package config

type OpWeights struct {
	Head float64 `yaml:"head"`
	Get  float64 `yaml:"get"`
	Post float64 `yaml:"post"`
}

type Asymmetric struct {
	Enabled                      bool      `yaml:"enabled"`
	OperationWeights             OpWeights `yaml:"operation_weights"`
	DownloadMaxBytes             int64     `yaml:"download_max_bytes"`
	UploadMaxBytes               int64     `yaml:"upload_max_bytes"`
	TargetRxTxRatio              float64   `yaml:"target_rx_tx_ratio"`
	RatioAdjustIntervalSec       int       `yaml:"ratio_adjust_interval_seconds"`
	MaxConcurrentLargeDownloads  int       `yaml:"max_concurrent_large_downloads"`
	GetPath                      string    `yaml:"get_path"`
	PostPath                     string    `yaml:"post_path"`
	GlobalQPSLarge               float64   `yaml:"global_qps_large"`
	ReceiveBytesPerSecond        float64   `yaml:"receive_bytes_per_second"`
	SendBytesPerSecond           float64   `yaml:"send_bytes_per_second"`
	MaxRedirects                 int       `yaml:"max_redirects"`
	TransportMaxIdleConnsPerHost int       `yaml:"transport_max_idle_conns_per_host"`
	TransportIdleConnTimeoutSec  int       `yaml:"transport_idle_conn_timeout_seconds"`
	TotalDownloadCapBytes        uint64    `yaml:"total_download_cap_bytes"`
	TotalUploadCapBytes          uint64    `yaml:"total_upload_cap_bytes"`
	HeadEstimateRxBytes          int64     `yaml:"head_estimate_rx_bytes"`
	HeadEstimateTxBytes          int64     `yaml:"head_estimate_tx_bytes"`
}
