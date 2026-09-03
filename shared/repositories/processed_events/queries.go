package processed_events

import (
	"embed"
	"shared/repositories/sqlutils"
)

//go:embed queries/*.sql
var queryFiles embed.FS

type queries struct {
	insertProcessedEvent string
	insertNotification   string
}

func mustLoadQueries() *queries {
	return &queries{
		insertProcessedEvent: sqlutils.MustLoadQuery(queryFiles, "queries/insert_processed_event.sql"),
		insertNotification:   sqlutils.MustLoadQuery(queryFiles, "queries/insert_notification.sql"),
	}
}
