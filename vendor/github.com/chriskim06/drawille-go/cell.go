package drawille

const (
	BRAILLE_OFFSET = '\u2800'
	LINE_OFFSET    = '\u2500'
	NO_OFFSET      = '\u0000'
)

var BRAILLE = [4][2]rune{
	{'\u0001', '\u0008'},
	{'\u0002', '\u0010'},
	{'\u0004', '\u0020'},
	{'\u0040', '\u0080'},
}

const (
	YAXIS        = '\u0024' // ┤
	XAXIS        = '\u0000' // ─
	ORIGIN       = '\u0070' //╰
	XLABELMARKER = '\u002C' // ┬
	LABELSTART   = '\u0014' // └
	LABELEND     = '\u0018' // ┘
)

// Cell represents the braille character at some coordinate in the canvas
type Cell struct {
	val    rune
	offset rune
	color  Color
}

func NewCell(r, offset rune, color Color) Cell {
	return Cell{
		val:    r,
		offset: offset,
		color:  color,
	}
}

// String returns the cell's rune wrapped in the color escape strings
func (c Cell) String() string {
	if c.val+c.offset == 0 {
		return " "
	}
	return wrap(string(c.val+c.offset), c.color)
}
