//go:build ! darwin

package gui

import (
	"bytes"
	"image"
	"regexp"
	"sync/atomic"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/label"
	"github.com/aarzilli/nucular/font"
	"github.com/aarzilli/nucular/style"
	"github.com/rocketsoftware/open-web-launch/utils"
	"github.com/rocketsoftware/open-web-launch/utils/log"
	"golang.org/x/mobile/event/key"
)

func New() *GUI {
	gui := &GUI{}
	gui.ready = make(chan (interface{}))
	gui.title.Store("")
	gui.text.Store("")
	gui.progressMax.Store(0)
	return gui
}

func (gui *GUI) Start(windowTitle string) error {
	if gui == nil {
		return nil
	}
	imageBytes, err := Asset("assets/Icon64.png")
	if err != nil {
		return err
	}
	reader := bytes.NewReader(imageBytes)
	img, err := utils.LoadPngImage(reader)
	if err != nil {
		return err
	}
	gui.icon = img
	go func() {
		gui.WaitForWindow()
		if err := utils.LoadIconAndSetForWindow(windowTitle); err != nil {
			log.Printf("warning: unable to set window icon: %v", err)
		}
	}()
	window := nucular.NewMasterWindowSize(0, windowTitle, image.Point{470, 240}, gui.updateFn)
	window.SetStyle(gui.makeStyle())
	gui.window = window
	window.Main()
	return nil
}

func (gui *GUI) makeStyle() *style.Style {
	style := style.FromTable(myThemeTable, scaling)
	style.Button.TextActive = myThemeTable.ColorBorder
	style.Button.TextNormal = myThemeTable.ColorBorder
	style.Button.Rounding = 0
	style.Button.Border = 1
	style.Button.Padding = image.Point{4, 4}
	style.Progress.Rounding = 0
	style.Progress.Padding = image.Point{0, 0}
	style.NormalWindow.Padding = image.Point{20, 0}
	myFont, err := utils.LoadFont("Arial", 11, scaling)
	if err != nil {
		log.Printf("warning: %v\n", err)
	}
	wrappedFont := font.Face{Face: myFont}
	style.Font = wrappedFont
	return style
}

func (gui *GUI) updateFn(w *nucular.Window) {
	gui.emitWindowReady()
	if w.Input().Keyboard.Pressed(key.CodeEscape) {
		log.Println("escape pressed, closing window...")
		gui.SendTextMessage("Cancelling...")
		go gui.cancel(w)
	}
	centralPartWidth := 420
	iconWidth := 80
	textWidth := centralPartWidth - iconWidth

	w.Row(20).Dynamic(1)
	w.Spacing(1)

	w.Row(64).Static(iconWidth, textWidth)
	w.Image(gui.icon)
	w.Label(gui.title.Load().(string), "CC")

	if gui.err != nil {
		re := regexp.MustCompile("[^!-~\t ]")
		text := re.ReplaceAllLiteralString(gui.err.Error(), "")
		var extraLine string
		if err, ok := gui.err.(*utils.ErrorWithExtraLine); ok {
			extraLine = err.ExtraLine()
		}
		if extraLine != "" {
			w.Row(20).Dynamic(1)
			w.Label(text, "LT")
			w.Row(65).Dynamic(1)
			w.Label(extraLine, "LT")
		} else {
			w.Row(85).Dynamic(1)
			w.LabelWrap(text)
		}


		w.Row(30).Dynamic(5)
		if gui.logFile != "" {
			w.Spacing(3)
			if w.Button(label.TA("Open Log", "CC"), false) {
				gui.openLog()
			}
		} else {
			w.Spacing(4)
		}
		if w.Button(label.TA("Close", "CC"), false) {
			log.Println("close button pressed, closing window...")
			go gui.cancel(w)
		}
		return
	}
	w.Row(30).Dynamic(1)
	w.Spacing(1)

	w.Row(20).Dynamic(1)
	w.Label(gui.text.Load().(string), "LC")

	w.Row(12).Dynamic(1)
	progress := int(atomic.LoadInt32(&gui.progress))
	progressMax := gui.progressMax.Load().(int)
	w.Progress(&progress, progressMax, false)

	w.Row(10).Dynamic(1)
	w.Spacing(1)

	w.Row(30).Dynamic(5)
	w.Spacing(4)
	if progress < progressMax {
		if w.Button(label.TA("Cancel", "CC"), false) {
			log.Println("cancel button pressed, closing window...")
			go gui.cancel(w)
		}
	} else {
		if w.Button(label.TA("Close", "CC"), false) {
			log.Println("close button pressed, closing window...")
			go gui.cancel(w)
		}
	}
}

func (gui *GUI) emitWindowReady() {
	gui.readyOnce.Do(func() { close(gui.ready) })
}


func (gui *GUI) cancel(w *nucular.Window) {
	w.Master().Close()
}

func (gui *GUI) SendTextMessage(text string) error {
	if gui == nil {
		return nil
	}
	gui.text.Store(text)
	gui.window.Changed()
	return nil
}

func (gui *GUI) SendErrorMessage(err error) error {
	if gui == nil {
		return nil
	}
	gui.err = err
	gui.window.Changed()
	return nil
}

func (gui *GUI) SendCloseMessage() error {
	if gui == nil {
		return nil
	}
	gui.window.Close()
	return nil
}

func (gui *GUI) SetTitle(title string) error {
	if gui == nil {
		return nil
	}
	gui.title.Store(title)
	gui.window.Changed()
	return nil
}

func (gui *GUI) SetProgressMax(val int) {
	if gui == nil {
		return
	}
	gui.progressMax.Store(val)
}

func (gui *GUI) ProgressStep() {
	if gui == nil {
		return
	}
	atomic.AddInt32(&gui.progress, 1)
	gui.window.Changed()
}



func (gui *GUI) openLog() {
	if gui == nil {
		return
	}
	if gui.logFile == "" {
		return
	}
	utils.OpenTextFile(gui.logFile)
}

func (gui *GUI) Closed() bool {
	if gui == nil {
		return false
	}
	return gui.window.Closed()
}

func (gui *GUI) Terminate() error {
	if gui == nil {
		return nil
	}
	if gui.window != nil {
		if !gui.window.Closed() {
			go gui.window.Close()
		}
	}
	return nil
}
