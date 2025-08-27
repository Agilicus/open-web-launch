package gui

import (
	"image"
	"image/color"
	"sync"
	"sync/atomic"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/style"
)

const scaling = 1.2

type GUI struct {
	windowTitle string
	title       atomic.Value
	text        atomic.Value
	progress    int32
	progressMax atomic.Value
	window      nucular.MasterWindow
	ready       chan interface{} // the channel closed when the gui appears for the first time
	readyOnce   sync.Once        // protects ready channel from being closed twice
	icon        *image.RGBA
	err         error
	logFile     string
}

// myThemeTable is modified WhiteTheme
var myThemeTable = style.ColorTable{
	ColorText:                  color.RGBA{0x3c, 0x3c, 0x3c, 255}, // modified
	ColorWindow:                color.RGBA{255, 255, 255, 255},    // modified
	ColorHeader:                color.RGBA{175, 175, 175, 255},
	ColorHeaderFocused:         color.RGBA{0xc3, 0x9a, 0x9a, 255},
	ColorBorder:                color.RGBA{0x25, 0x59, 0xA9, 255}, // modified
	ColorButton:                color.RGBA{255, 255, 255, 255},    // modified
	ColorButtonHover:           color.RGBA{255, 255, 255, 255},    // modified
	ColorButtonActive:          color.RGBA{255, 255, 255, 255},    // modified
	ColorToggle:                color.RGBA{150, 150, 150, 255},
	ColorToggleHover:           color.RGBA{120, 120, 120, 255},
	ColorToggleCursor:          color.RGBA{175, 175, 175, 255},
	ColorSelect:                color.RGBA{175, 175, 175, 255},
	ColorSelectActive:          color.RGBA{190, 190, 190, 255},
	ColorSlider:                color.RGBA{0xCE, 0xCE, 0xCE, 255}, // modified
	ColorSliderCursor:          color.RGBA{0x25, 0x59, 0xA9, 255}, // modified
	ColorSliderCursorHover:     color.RGBA{70, 70, 70, 255},
	ColorSliderCursorActive:    color.RGBA{60, 60, 60, 255},
	ColorProperty:              color.RGBA{175, 175, 175, 255},
	ColorEdit:                  color.RGBA{150, 150, 150, 255},
	ColorEditCursor:            color.RGBA{0, 0, 0, 255},
	ColorCombo:                 color.RGBA{175, 175, 175, 255},
	ColorChart:                 color.RGBA{160, 160, 160, 255},
	ColorChartColor:            color.RGBA{45, 45, 45, 255},
	ColorChartColorHighlight:   color.RGBA{255, 0, 0, 255},
	ColorScrollbar:             color.RGBA{180, 180, 180, 255},
	ColorScrollbarCursor:       color.RGBA{140, 140, 140, 255},
	ColorScrollbarCursorHover:  color.RGBA{150, 150, 150, 255},
	ColorScrollbarCursorActive: color.RGBA{160, 160, 160, 255},
	ColorTabHeader:             color.RGBA{180, 180, 180, 255},
}


func (gui *GUI) WaitForWindow() {
	if gui == nil {
		return
	}
	<-gui.ready
}

func (gui *GUI) SetLogFile(logFile string) {
	if gui == nil {
		return
	}
	gui.logFile = logFile
}