package drawille

import (
	"fmt"
	"image"
	"math"
	"strings"
)

// Canvas is the container for a series of plots
// Vertical labels are shown by default and a starting
// and ending horizontal label can optionally be shown
type Canvas struct {
	// various settings accessible outside this object
	LineColors []Color
	LabelColor Color
	AxisColor  Color
	ShowAxis   bool

	// a list of labels the canvas will print for the x axis
	// horizontal labels must be provided by the caller. too lazy
	// to come up with a good way to print stuff so offloading some
	// of that work to the user. when the horizontal labels arent
	// provided an empty line is printed
	HorizontalLabels []string

	// this value is used to determine the horizontal scale when
	// graphing points in the plot. with braille characters, each
	// cell represents 2 points along the x axis. so if the total
	// graphable area in the canvas is 50 cells then we can plot
	// 100 points of data. if we only want to plot 50 data points
	// then the horizontal scale needs to be 2 in order to utilize
	// the whole graphable area. since image uses ints for points
	// the graphing gets a little weird at times due to rounding.
	NumDataPoints int

	// the bounds of the canvas
	area image.Rectangle

	// the width of the graphable area in terms of braille dots
	// so 2 * (rectangle width - offset)
	plotWidth int

	// the number of usable rows in the canvas. this is decremented
	// when the x axis is drawn and when there are horizontal labels
	graphHeight int

	// a map of the entire braille grid
	points map[image.Point]Cell

	// number of columns from the left the plot starts at. this is the
	// width of the y axis and the longest label
	horizontalOffset int

	// this is the furthest column that has been plotted in the canvas
	// used to help with the horizontal labels
	maxX int
}

// NewCanvas creates a default canvas
func NewCanvas(width, height int) Canvas {
	c := Canvas{
		AxisColor:  Default,
		LabelColor: Default,
		LineColors: []Color{},
		ShowAxis:   true,
		area:       image.Rect(0, 0, width, height),
		points:     make(map[image.Point]Cell),
	}
	return c
}

// Fill adds values to the Canvas
func (c *Canvas) Fill(data [][]float64) {
	if len(data) == 0 {
		return
	}
	c.points = make(map[image.Point]Cell)
	c.graphHeight = c.area.Dy()
	minDataPoint, maxDataPoint := getMinMaxFloat64From2dSlice(data)
	diff := maxDataPoint - minDataPoint

	// y axis
	if c.ShowAxis {
		c.graphHeight--
		if len(c.HorizontalLabels) != 0 {
			c.graphHeight--
		}
		lenMaxDataPoint := len(fmt.Sprintf("%.2f", maxDataPoint))
		lenMinDataPoint := len(fmt.Sprintf("%.2f", minDataPoint))
		if lenMinDataPoint > lenMaxDataPoint {
			lenMaxDataPoint = lenMinDataPoint
		}
		c.horizontalOffset = lenMaxDataPoint + 2 // y-axis plus space before it
		verticalScale := diff / float64(c.graphHeight-1)
		cur := minDataPoint
		for i := c.graphHeight - 1; i >= 0; i-- {
			val := fmt.Sprintf("%.2f", cur)
			c.setRunes(i, lenMaxDataPoint-len(val), c.LabelColor, NO_OFFSET, []rune(val)...)
			c.setRunes(i, lenMaxDataPoint+1, c.AxisColor, LINE_OFFSET, YAXIS)
			cur += verticalScale
		}
	}
	c.plotWidth = (c.area.Dx() - c.horizontalOffset) * 2

	// plot the data
	horizontalScale := 1.0
	if c.NumDataPoints > 0 {
		horizontalScale = math.Round(float64(c.plotWidth/c.NumDataPoints) + 0.5)
	}
	for i, line := range data {
		if len(line) == 0 {
			continue
		} else if c.NumDataPoints > 0 && len(line) > c.NumDataPoints {
			line = line[len(line)-c.NumDataPoints:]
		} else if len(line) > c.plotWidth {
			line = line[len(line)-c.plotWidth:]
		}

		// y coordinates are calculated as percentages of the graph height. the percentage
		// is the current point minus the smallest point in the dataset over the diff
		// between largest and smallest points. this means that with small graphs there
		// can be some squashing of the graph due to rounding.
		previousHeight := int(((line[0] - minDataPoint) / diff) * float64(c.graphHeight-1))
		for j, val := range line[1:] {
			height := int(((val - minDataPoint) / diff) * float64(c.graphHeight-1))
			c.setLine(
				image.Pt(
					(c.horizontalOffset*2)+int(float64(j)*horizontalScale),
					(c.graphHeight-1-previousHeight)*4,
				),
				image.Pt(
					(c.horizontalOffset*2)+int(float64(j+1)*horizontalScale),
					(c.graphHeight-1-height)*4,
				),
				c.lineColor(i),
			)
			previousHeight = height
		}
	}

	// x axis
	if c.ShowAxis {
		axisRunes := []rune{ORIGIN}
		remaining := c.plotWidth / 2
		if len(c.HorizontalLabels) != 0 && len(c.HorizontalLabels) <= remaining {
			// always print the starting label then figure out if there's enough space to print the ending label.
			// the length of the longest plot needs to be at least the length of the two labels plus 2 spaces
			// for padding between plus 2 for the unicode line characters printed next to them.
			start := c.HorizontalLabels[0]
			end := c.HorizontalLabels[len(c.HorizontalLabels)-1]
			c.setRunes(c.graphHeight+1, c.horizontalOffset, c.AxisColor, LINE_OFFSET, LABELSTART)
			c.setRunes(c.graphHeight+1, c.horizontalOffset+1, c.LabelColor, NO_OFFSET, []rune(start)...)
			axisRunes = append(axisRunes, XLABELMARKER)
			remaining--
			minWidth := len(start) + len(end) + 4
			if length := c.maxX - c.horizontalOffset; length >= minWidth {
				labelPos := c.horizontalOffset + length - len(end)
				c.setRunes(c.graphHeight+1, labelPos, c.LabelColor, NO_OFFSET, []rune(end)...)
				c.setRunes(c.graphHeight+1, labelPos+len(end), c.AxisColor, LINE_OFFSET, LABELEND)
				axisRunes = append(axisRunes, repeatRune(XAXIS, length-1)...)
				axisRunes = append(axisRunes, XLABELMARKER)
				remaining -= length
			}
		}
		axisRunes = append(axisRunes, repeatRune(XAXIS, remaining)...)
		c.setRunes(c.graphHeight, c.horizontalOffset-1, c.AxisColor, LINE_OFFSET, axisRunes...)
	}
}

// Plot sets the Canvas and return the string representation of it
func (c *Canvas) Plot(data [][]float64) string {
	if len(data) == 0 {
		return ""
	}
	c.Fill(data)
	return c.String()
}

// String allows the Canvas to implement the Stringer interface
func (c Canvas) String() string {
	if len(c.points) == 0 {
		return ""
	}

	// go through each row of the canvas and print the lines
	var b strings.Builder
	for row := 0; row < c.area.Dy(); row++ {
		for col := 0; col < c.area.Dx(); col++ {
			if cell, ok := c.points[image.Pt(col, row)]; ok {
				b.WriteString(cell.String())
			} else {
				b.WriteString(" ")
			}
		}
		if row < c.area.Dy()-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (c *Canvas) setRunes(row, col int, color Color, offset rune, runes ...rune) {
	for i, r := range runes {
		if p := image.Pt(col+i, row); p.In(c.area) {
			c.points[p] = NewCell(r, offset, color)
		}
	}
}

func (c *Canvas) setLine(p0, p1 image.Point, color Color) {
	for _, pt := range line(p0, p1) {
		if p := image.Pt(pt.X/2, pt.Y/4); p.In(c.area) {
			if p.X > c.maxX {
				c.maxX = p.X
			}
			c.points[p] = NewCell(c.points[p].val|BRAILLE[pt.Y%4][pt.X%2], BRAILLE_OFFSET, color)
		}
	}
}

func (c Canvas) lineColor(i int) Color {
	if len(c.LineColors) <= 0 || i >= len(c.LineColors) {
		return Default
	}
	return c.LineColors[i]
}
