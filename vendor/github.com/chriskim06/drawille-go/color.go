package drawille

import "fmt"

type Color int

var (
	reset                Color = -2
	Default              Color = -1
	AliceBlue            Color = 255
	AntiqueWhite         Color = 255
	Aqua                 Color = 14
	Aquamarine           Color = 122
	Azure                Color = 15
	Beige                Color = 230
	Bisque               Color = 224
	Black                Color = 0
	BlanchedAlmond       Color = 230
	Blue                 Color = 12
	BlueViolet           Color = 92
	Brown                Color = 88
	BurlyWood            Color = 180
	CadetBlue            Color = 73
	Chartreuse           Color = 118
	Chocolate            Color = 166
	Coral                Color = 209
	CornflowerBlue       Color = 68
	Cornsilk             Color = 230
	Crimson              Color = 161
	Cyan                 Color = 14
	DarkBlue             Color = 18
	DarkCyan             Color = 30
	DarkGoldenrod        Color = 136
	DarkGray             Color = 248
	DarkGreen            Color = 22
	DarkKhaki            Color = 143
	DarkMagenta          Color = 90
	DarkOliveGreen       Color = 59
	DarkOrange           Color = 208
	DarkOrchid           Color = 134
	DarkRed              Color = 88
	DarkSalmon           Color = 173
	DarkSeaGreen         Color = 108
	DarkSlateBlue        Color = 60
	DarkSlateGray        Color = 238
	DarkTurquoise        Color = 44
	DarkViolet           Color = 92
	DeepPink             Color = 198
	DeepSkyBlue          Color = 39
	DimGray              Color = 242
	DodgerBlue           Color = 33
	Firebrick            Color = 124
	FloralWhite          Color = 15
	ForestGreen          Color = 28
	Fuchsia              Color = 13
	Gainsboro            Color = 253
	GhostWhite           Color = 15
	Gold                 Color = 220
	Goldenrod            Color = 178
	Gray                 Color = 8
	Green                Color = 2
	GreenYellow          Color = 155
	Honeydew             Color = 15
	HotPink              Color = 205
	IndianRed            Color = 167
	Indigo               Color = 54
	Ivory                Color = 15
	Khaki                Color = 222
	Lavender             Color = 254
	LavenderBlush        Color = 255
	LawnGreen            Color = 118
	LemonChiffon         Color = 230
	LightBlue            Color = 152
	LightCoral           Color = 210
	LightCyan            Color = 195
	LightGoldenrodYellow Color = 230
	LightGray            Color = 252
	LightGreen           Color = 120
	LightPink            Color = 217
	LightSalmon          Color = 216
	LightSeaGreen        Color = 37
	LightSkyBlue         Color = 117
	LightSlateGray       Color = 103
	LightSteelBlue       Color = 152
	LightYellow          Color = 230
	Lime                 Color = 10
	LimeGreen            Color = 77
	Linen                Color = 255
	Magenta              Color = 13
	Maroon               Color = 1
	MediumAquamarine     Color = 79
	MediumBlue           Color = 20
	MediumOrchid         Color = 134
	MediumPurple         Color = 98
	MediumSeaGreen       Color = 72
	MediumSlateBlue      Color = 99
	MediumSpringGreen    Color = 48
	MediumTurquoise      Color = 80
	MediumVioletRed      Color = 162
	MidnightBlue         Color = 17
	MintCream            Color = 15
	MistyRose            Color = 224
	Moccasin             Color = 223
	NavajoWhite          Color = 223
	Navy                 Color = 4
	OldLace              Color = 230
	Olive                Color = 3
	OliveDrab            Color = 64
	Orange               Color = 214
	OrangeRed            Color = 202
	Orchid               Color = 170
	PaleGoldenrod        Color = 223
	PaleGreen            Color = 120
	PaleTurquoise        Color = 159
	PaleVioletRed        Color = 168
	PapayaWhip           Color = 230
	PeachPuff            Color = 223
	Peru                 Color = 173
	Pink                 Color = 218
	Plum                 Color = 182
	PowderBlue           Color = 152
	Purple               Color = 5
	Red                  Color = 9
	RosyBrown            Color = 138
	RoyalBlue            Color = 63
	SaddleBrown          Color = 94
	Salmon               Color = 210
	SandyBrown           Color = 215
	SeaGreen             Color = 29
	SeaShell             Color = 15
	Sienna               Color = 131
	Silver               Color = 7
	SkyBlue              Color = 117
	SlateBlue            Color = 62
	SlateGray            Color = 66
	Snow                 Color = 15
	SpringGreen          Color = 48
	SteelBlue            Color = 67
	Tan                  Color = 180
	Teal                 Color = 6
	Thistle              Color = 182
	Tomato               Color = 203
	Turquoise            Color = 80
	Violet               Color = 213
	Wheat                Color = 223
	White                Color = 15
	WhiteSmoke           Color = 255
	Yellow               Color = 11
	YellowGreen          Color = 149
)

func (c Color) String() string {
	if c == reset || c == Default {
		return "\x1b[0m"
	}
	return fmt.Sprintf("\x1b[38;5;%dm", c)
}
