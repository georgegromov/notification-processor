package sqlutils

import (
	"embed"
	"fmt"
)

func MustLoadQuery(fs embed.FS, filepath string) string {
	query, err := fs.ReadFile(filepath)
	if err != nil {
		panic(fmt.Errorf("failed to read query file %s: %w", filepath, err))
	}
	return string(query)
}
