package pb

import (
	"fmt"
	"time"
)

type Units int

const (
	// U_NO are default units, they represent a simple value and are not formatted at all.
	U_NO Units = iota
	// U_BYTES units are formatted in a human readable way (b, Bb, Mb, ...)
	U_BYTES
	// U_DURATION units are formatted in a human readable way (3h14m15s)
	U_DURATION
)

const (
	KiB = 1024
	MiB = 1048576
	GiB = 1073741824
	TiB = 1099511627776
)

func Format(i int64) *formatter {
	return &formatter{n: i}
}

type formatter struct {
	n      int64
	unit   Units
	width  int
	perSec bool
}

func (f *formatter) To(unit Units) *formatter {
	f.unit = unit
	return f
}

func (f *formatter) Width(width int) *formatter {
	f.width = width
	return f
}

func (f *formatter) PerSec() *formatter {
	f.perSec = true
	return f
}

func (f *formatter) String() (out string) {
	switch f.unit {
	case U_BYTES:
		out = formatBytes(f.n)
	case U_DURATION:
		out = formatDuration(f.n)
	default:
		out = fmt.Sprintf(fmt.Sprintf("%%%dd", f.width), f.n)
	}
	if f.perSec {
		out += "/s"
	}
	return
}

// Convert bytes to human readable string. Like a 2 MiB, 64.2 KiB, 52 B
func formatBytes(i int64) (result string) {
	switch {
	case i >= TiB:
		result = fmt.Sprintf("%.02f TiB", float64(i)/TiB)
	case i >= GiB:
		result = fmt.Sprintf("%.02f GiB", float64(i)/GiB)
	case i >= MiB:
		result = fmt.Sprintf("%.02f MiB", float64(i)/MiB)
	case i >= KiB:
		result = fmt.Sprintf("%.02f KiB", float64(i)/KiB)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	return
}

func formatDuration(n int64) (result string) {
	d := time.Duration(n)
	if d > time.Hour*24 {
		result = fmt.Sprintf("%dd", d/24/time.Hour)
		d -= (d / time.Hour / 24) * (time.Hour * 24)
	}
	result = fmt.Sprintf("%s%v", result, d)
	return
}
