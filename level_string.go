// generated by stringer -type=Level; DO NOT EDIT

package vlog

import "fmt"

const _Level_name = "v2v1infoerr"

var _Level_index = [...]uint8{2, 4, 8, 11}

func (i Level) String() string {
	i -= -2
	if i < 0 || i >= Level(len(_Level_index)) {
		return fmt.Sprintf("Level(%d)", i+-2)
	}
	hi := _Level_index[i]
	lo := uint8(0)
	if i > 0 {
		lo = _Level_index[i-1]
	}
	return _Level_name[lo:hi]
}
