# drawille-go

started out as a fork of [exrook/drawille-go](https://github.com/exrook/drawille-go) but has diverged a decent amount from that. a lot of the changes have been adapted from [termui's implementation](https://github.com/gizak/termui/blob/master/drawille/drawille.go) but modified to try to incorporate the x and y axis plus their labels.

this uses golang's image library (specifically the Rectangle and Point types) for the canvas that data is plotted in. each point in the canvas is a rune with its own color that gets returned as a string so that it can be used in terminal applications.
