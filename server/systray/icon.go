//go:build gui

package systray

import (
	_ "embed"
)

//go:embed icon.png
var iconData []byte

func getIcon() []byte {
	return iconData
}
