package processed_events

import (
	"embed"
	"shared/repositories/sqlutils"
)

var queryFiles embed.FS

type queries struct {
	insertProcessedEvent string
	insertNotification   string
}

func mustLoadQueries() *queries {
	return &queries{
		insertProcessedEvent: sqlutils.MustLoadQuery(queryFiles, "insert_processed_event.sql"),
		insertNotification:   sqlutils.MustLoadQuery(queryFiles, "insert_notification.sql"),
	}
}
