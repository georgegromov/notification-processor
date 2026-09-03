package notifications

import (
	"embed"

	"shared/repositories/sqlutils"
)

//go:embed queries/*.sql
var queryFiles embed.FS

type queries struct {
	claimPending  string
	markSent      string
	scheduleRetry string
	markFailed    string
	reclaimStale  string
}

func mustLoadQueries() *queries {
	return &queries{
		claimPending:  sqlutils.MustLoadQuery(queryFiles, "queries/claim_pending.sql"),
		markSent:      sqlutils.MustLoadQuery(queryFiles, "queries/mark_sent.sql"),
		scheduleRetry: sqlutils.MustLoadQuery(queryFiles, "queries/schedule_retry.sql"),
		markFailed:    sqlutils.MustLoadQuery(queryFiles, "queries/mark_failed.sql"),
		reclaimStale:  sqlutils.MustLoadQuery(queryFiles, "queries/reclaim_stale.sql"),
	}
}
