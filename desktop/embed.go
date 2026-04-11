//go:build desktop

package desktop

import "embed"

//go:embed all:frontend/dist
var Assets embed.FS
