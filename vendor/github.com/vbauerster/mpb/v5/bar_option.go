package mpb

import (
	"bytes"
	"io"

	"github.com/vbauerster/mpb/v5/decor"
)

// BarOption is a function option which changes the default behavior of a bar.
type BarOption func(*bState)

func (s *bState) addDecorators(dest *[]decor.Decorator, decorators ...decor.Decorator) {
	type mergeWrapper interface {
		MergeUnwrap() []decor.Decorator
	}
	for _, decorator := range decorators {
		if mw, ok := decorator.(mergeWrapper); ok {
			*dest = append(*dest, mw.MergeUnwrap()...)
		}
		*dest = append(*dest, decorator)
	}
}

// AppendDecorators let you inject decorators to the bar's right side.
func AppendDecorators(decorators ...decor.Decorator) BarOption {
	return func(s *bState) {
		s.addDecorators(&s.aDecorators, decorators...)
	}
}

// PrependDecorators let you inject decorators to the bar's left side.
func PrependDecorators(decorators ...decor.Decorator) BarOption {
	return func(s *bState) {
		s.addDecorators(&s.pDecorators, decorators...)
	}
}

// BarID sets bar id.
func BarID(id int) BarOption {
	return func(s *bState) {
		s.id = id
	}
}

// BarWidth sets bar width independent of the container.
func BarWidth(width int) BarOption {
	return func(s *bState) {
		s.width = width
	}
}

// BarQueueAfter queues this (being constructed) bar to relplace
// runningBar after it has been completed.
func BarQueueAfter(runningBar *Bar) BarOption {
	if runningBar == nil {
		return nil
	}
	return func(s *bState) {
		s.runningBar = runningBar
	}
}

// BarRemoveOnComplete removes both bar's filler and its decorators
// on complete event.
func BarRemoveOnComplete() BarOption {
	return func(s *bState) {
		s.dropOnComplete = true
	}
}

// BarFillerClearOnComplete clears bar's filler on complete event.
// It's shortcut for BarFillerOnComplete("").
func BarFillerClearOnComplete() BarOption {
	return BarFillerOnComplete("")
}

// BarFillerOnComplete replaces bar's filler with message, on complete event.
func BarFillerOnComplete(message string) BarOption {
	return func(s *bState) {
		s.filler = makeBarFillerOnComplete(s.baseF, message)
	}
}

func makeBarFillerOnComplete(filler BarFiller, message string) BarFiller {
	return BarFillerFunc(func(w io.Writer, width int, st *decor.Statistics) {
		if st.Completed {
			io.WriteString(w, message)
		} else {
			filler.Fill(w, width, st)
		}
	})
}

// BarPriority sets bar's priority. Zero is highest priority, i.e. bar
// will be on top. If `BarReplaceOnComplete` option is supplied, this
// option is ignored.
func BarPriority(priority int) BarOption {
	return func(s *bState) {
		s.priority = priority
	}
}

// BarExtender is an option to extend bar to the next new line, with
// arbitrary output.
func BarExtender(extender BarFiller) BarOption {
	if extender == nil {
		return nil
	}
	return func(s *bState) {
		s.extender = makeExtFunc(extender)
	}
}

func makeExtFunc(extender BarFiller) extFunc {
	buf := new(bytes.Buffer)
	nl := []byte("\n")
	return func(r io.Reader, tw int, st *decor.Statistics) (io.Reader, int) {
		extender.Fill(buf, tw, st)
		return io.MultiReader(r, buf), bytes.Count(buf.Bytes(), nl)
	}
}

// TrimSpace trims bar's edge spaces.
func TrimSpace() BarOption {
	return func(s *bState) {
		s.trimSpace = true
	}
}

// BarStyle overrides mpb.DefaultBarStyle which is "[=>-]<+".
// It's ok to pass string containing just 5 runes, for example "╢▌▌░╟",
// if you don't need to override '<' (reverse tip) and '+' (refill rune).
func BarStyle(style string) BarOption {
	if style == "" {
		return nil
	}
	type styleSetter interface {
		SetStyle(string)
	}
	return func(s *bState) {
		if t, ok := s.baseF.(styleSetter); ok {
			t.SetStyle(style)
		}
	}
}

// BarNoPop disables bar pop out of container. Effective when
// PopCompletedMode of container is enabled.
func BarNoPop() BarOption {
	return func(s *bState) {
		s.noPop = true
	}
}

// BarReverse reverse mode, bar will progress from right to left.
func BarReverse() BarOption {
	type revSetter interface {
		SetReverse(bool)
	}
	return func(s *bState) {
		if t, ok := s.baseF.(revSetter); ok {
			t.SetReverse(true)
		}
	}
}

// SpinnerStyle sets custom spinner style.
// Effective when Filler type is spinner.
func SpinnerStyle(frames []string) BarOption {
	if len(frames) == 0 {
		return nil
	}
	chk := func(filler BarFiller) (interface{}, bool) {
		t, ok := filler.(*spinnerFiller)
		return t, ok
	}
	cb := func(t interface{}) {
		t.(*spinnerFiller).frames = frames
	}
	return MakeFillerTypeSpecificBarOption(chk, cb)
}

// MakeFillerTypeSpecificBarOption makes BarOption specific to Filler's
// actual type. If you implement your own Filler, so most probably
// you'll need this. See BarStyle or SpinnerStyle for example.
func MakeFillerTypeSpecificBarOption(
	typeChecker func(BarFiller) (interface{}, bool),
	cb func(interface{}),
) BarOption {
	return func(s *bState) {
		if t, ok := typeChecker(s.baseF); ok {
			cb(t)
		}
	}
}

// BarOptOn returns option when condition evaluates to true.
func BarOptOn(option BarOption, condition func() bool) BarOption {
	if condition() {
		return option
	}
	return nil
}
