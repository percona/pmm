package models

type HAParams struct {
	GrafanaGossipPort int
	Enabled           bool
	Bootstrap         bool
	NodeID            string
	AdvertiseAddress  string
	Nodes             []string
	RaftPort          int
	GossipPort        int
}

type Params struct {
	HAParams *HAParams
	VMParams *VictoriaMetricsParams
	PGParams *PGParams
}
