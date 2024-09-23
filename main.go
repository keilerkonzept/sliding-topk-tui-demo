package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tui "github.com/charmbracelet/bubbletea"
	styles "github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	plot "github.com/chriskim06/drawille-go"
	"github.com/keilerkonzept/topk"
	"github.com/keilerkonzept/topk/heap"
	"github.com/keilerkonzept/topk/sliding"
)

type Config struct {
	// sketch
	K            int
	Width        int
	Depth        int
	Decay        float64
	DecayLUTSize int
	TickSize     time.Duration
	WindowSize   time.Duration

	// render
	PlotFPS       int
	ItemsFPS      int
	ItemCountsFPS int
	TrackSelected bool
	LogScale      bool
	ViewSplit     int

	// input
	JSON            bool
	TimestampLayout string
}

var config = Config{
	K:            50,
	Width:        3000,
	Depth:        3,
	Decay:        0.9,
	DecayLUTSize: 8192,
	TickSize:     time.Second,
	WindowSize:   10 * time.Second,

	ViewSplit:     50,
	PlotFPS:       20,
	ItemsFPS:      1,
	ItemCountsFPS: 5,

	JSON:            false,
	TimestampLayout: time.RFC3339,
}

var (
	selectedColor = styles.AdaptiveColor{Light: "0", Dark: "14"}
	borderColor   = styles.AdaptiveColor{Light: "#555", Dark: "#555"}
	selectedFg    = styles.NewStyle().Foreground(selectedColor)
	borderFg      = styles.NewStyle().Foreground(borderColor)
	plotStyle     = styles.NewStyle().
			BorderStyle(styles.NormalBorder()).
			Foreground(borderColor).
			BorderForeground(borderColor)
)

func main() {
	log.SetOutput(os.Stdout)
	flag.IntVar(&config.K, "k", config.K, "Track the top K items")
	flag.IntVar(&config.Width, "width", config.Width, "Sketch width")
	flag.IntVar(&config.Depth, "depth", config.Depth, "Sketch depth")
	flag.DurationVar(&config.WindowSize, "window", config.WindowSize, "Window size")
	flag.DurationVar(&config.TickSize, "tick", config.TickSize, "Sliding window tick size (time bucket precision)")
	flag.Float64Var(&config.Decay, "decay", config.Decay, "Counter decay probability on collisions")
	flag.IntVar(&config.DecayLUTSize, "decay-lut-size", config.DecayLUTSize, "Sketch decay look-up table size")
	flag.IntVar(&config.PlotFPS, "plot-fps", config.PlotFPS, "Plot refresh rate (frames per second)")
	flag.IntVar(&config.ItemsFPS, "items-fps", config.ItemsFPS, "Item refresh rate (frames per second)")
	flag.IntVar(&config.ItemCountsFPS, "item-counts-fps", config.ItemCountsFPS, "Item counts refresh rate (frames per second)")
	flag.BoolVar(&config.JSON, "json", config.JSON, "Read JSON records {item,[count],[timestamp]} instead of text lines")
	flag.BoolVar(&config.TrackSelected, "track-selected", config.TrackSelected, "Keep the selected item focused")
	flag.BoolVar(&config.LogScale, "log-scale", config.LogScale, "Use a logarithmic Y axis scale (default: linear)")
	flag.StringVar(&config.TimestampLayout, "json-timestamp-layout", config.TimestampLayout, "Layout for string values of the timestamp field")
	flag.IntVar(&config.ViewSplit, "view-split", config.ViewSplit, "Split the view at this % of the total screen width [20,80]")
	flag.Parse()

	config.ViewSplit = max(20, config.ViewSplit)
	config.ViewSplit = min(80, config.ViewSplit)

	sketch := sliding.New(config.K,
		int(config.WindowSize/config.TickSize),
		sliding.WithWidth(config.Width),
		sliding.WithDepth(config.Depth),
		sliding.WithDecay(float32(config.Decay)),
		sliding.WithDecayLUTSize(config.DecayLUTSize),
	)

	m := newModel(sketch)
	if _, err := tui.NewProgram(m, tui.WithInputTTY()).Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	width, height int

	track    bool
	logScale atomic.Bool

	list         list.Model
	listStyle    styles.Style
	listDelegate *list.DefaultDelegate
	help         help.Model
	plot         *plot.Canvas

	sketch         *sliding.Sketch
	sketchMu       sync.Mutex
	plotData       [][]float64
	plotLineColors []plot.Color
	listItems      []heap.Item
	latestTick     time.Time

	timestampsFromData atomic.Bool

	mu sync.Mutex
}

func newModel(sketch *sliding.Sketch) *model {
	const (
		defaultWidth  = 80
		defaultHeight = 20
	)

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = styles.NewStyle().
		Border(styles.NormalBorder(), false, false, false, true).
		BorderForeground(borderColor).
		Foreground(selectedColor).
		Bold(false).
		Padding(0, 0, 0, 1)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.
		Foreground(selectedColor)
	d.ShowDescription = true

	l := list.New(make([]list.Item, 0), d, defaultWidth/2-2, defaultHeight)
	l.Styles.NoItems = l.Styles.NoItems.
		Padding(0, 2)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	p := plot.NewCanvas(defaultWidth, defaultHeight)
	p.NumDataPoints = sketch.BucketHistoryLength
	p.ShowAxis = false
	p.LineColors = make([]plot.Color, config.K+1)

	help := help.New()

	m := &model{
		track:          config.TrackSelected,
		sketch:         sketch,
		help:           help,
		list:           l,
		listDelegate:   &d,
		plot:           &p,
		plotData:       make([][]float64, config.K+1),
		plotLineColors: make([]plot.Color, config.K+1),
	}
	m.timestampsFromData.Store(true)
	m.logScale.Store(config.LogScale)
	for i := range m.plotData {
		m.plotData[i] = make([]float64, m.sketch.BucketHistoryLength)
	}
	m.plot.Fill(m.plotData)
	return m
}

func (m *model) leftWidth() int {
	return m.width * config.ViewSplit / 100
}
func (m *model) rightWidth() int {
	return m.width * (100 - config.ViewSplit) / 100
}
func (m *model) readAndCountInput() tui.Cmd {
	if term.IsTerminal(os.Stdin.Fd()) {
		return nil // no data on stdin
	}
	return func() tui.Msg {
		switch {
		case config.JSON:
			m.readJSONItems()
		default:
			m.readTextItems()
		}
		return nil
	}
}

func (m *model) readTextItems() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		m.sketchMu.Lock()
		m.sketch.Incr(scanner.Text())
		m.sketchMu.Unlock()
	}
}

func (m *model) readJSONItems() {
	var item struct {
		Item      string `json:"item"`
		Count     int    `json:"count"`
		Timestamp any    `json:"timestamp"`
	}
	dec := json.NewDecoder(bufio.NewReader(os.Stdin))
	var last time.Time
	for {
		err := dec.Decode(&item)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}
		if item.Timestamp == nil {
			m.timestampsFromData.Store(false)
		}
		if m.timestampsFromData.Load() {
			var t time.Time
			switch timestamp := item.Timestamp.(type) {
			case int:
				t = time.Unix(int64(timestamp), 0)
			case float64:
				t = time.Unix(int64(timestamp), 0)
			case string:
				t, _ = time.Parse(config.TimestampLayout, timestamp)
			}
			if !t.IsZero() {
				last = m.doSketchTicks(t, last)
				m.mu.Lock()
				m.latestTick = last
				m.mu.Unlock()
			}
		}
		m.sketchMu.Lock()
		m.sketch.Add(item.Item, max(1, uint32(item.Count)))
		m.sketchMu.Unlock()
	}
}

func (m *model) sketchTickCmd() tui.Cmd {
	return func() tui.Msg {
		var last time.Time
		ticker := time.NewTicker(time.Duration(config.TickSize))
		for {
			select {
			case t := <-ticker.C:
				if config.JSON && m.timestampsFromData.Load() {
					continue
				}
				t = t.Truncate(config.TickSize)
				m.mu.Lock()
				m.latestTick = t
				m.mu.Unlock()
				last = m.doSketchTicks(t, last)
			}
		}
	}
}

func (m *model) doSketchTicks(t time.Time, last time.Time) time.Time {
	t = t.Truncate(config.TickSize)
	if last.IsZero() {
		last = t
		return last
	}
	if ticks := int(t.Sub(last) / config.TickSize); ticks > 0 {
		m.sketchMu.Lock()
		m.sketch.Ticks(ticks)
		m.sketchMu.Unlock()
		last = t
	}
	return last
}

type ItemsTickMsg time.Time

func doItemsTick() tui.Cmd {
	return tui.Every(time.Second/time.Duration(config.ItemsFPS), func(t time.Time) tui.Msg {
		return ItemsTickMsg(t)
	})
}

type ItemCountsTickMsg time.Time

func doItemCountsTick() tui.Cmd {
	if config.ItemCountsFPS == config.ItemsFPS {
		return nil
	}
	return tui.Every(time.Second/time.Duration(config.ItemCountsFPS), func(t time.Time) tui.Msg {
		return ItemCountsTickMsg(t)
	})
}

type PlotTickMsg time.Time

func doPlotTick() tui.Cmd {
	return tui.Every(time.Second/time.Duration(config.PlotFPS), func(t time.Time) tui.Msg {
		return PlotTickMsg(t)
	})
}

func (m *model) Init() tui.Cmd {
	return tui.Batch(m.sketchTickCmd(), m.readAndCountInput(), doPlotTick(), doItemsTick(), doItemCountsTick())
}

func (m *model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case ItemCountsTickMsg:
		m.updateListItemCountsFromSketch()
		m.list.Update(msg)
		cmdList := m.updateList(msg)
		return m, tui.Batch(cmdList, doItemCountsTick())
	case ItemsTickMsg:
		m.updateTopK()
		cmdList := m.updateList(msg)
		return m, tui.Batch(cmdList, doItemsTick())
	case PlotTickMsg:
		cmdPlot := m.updatePlot(msg)
		return m, tui.Batch(cmdPlot, doPlotTick())
	case tui.KeyMsg:
		switch {
		case key.Matches(msg, keys.Scale):
			m.logScale.Store(!m.logScale.Load())
			return m, nil
		case key.Matches(msg, keys.Track):
			m.toggleTracking()
			return m, nil
		case key.Matches(msg, keys.Quit):
			return m, tui.Quit
		}
	case tui.WindowSizeMsg:
		w, h := msg.Width, msg.Height
		m.width, m.height = w, h
		m.list.SetSize(m.leftWidth()-2, h-3)
		m.resizePlot(m.rightWidth()-2, h-4)
		m.listStyle = styles.NewStyle().
			BorderStyle(styles.NormalBorder()).
			BorderForeground(borderColor).
			Height(m.list.Height()).
			Width(m.list.Width())
		return m, nil
	}

	cmdList := m.updateList(msg)
	return m, tui.Batch(cmdList)
}

func (m *model) toggleTracking() {
	m.mu.Lock()
	m.track = !m.track
	m.mu.Unlock()
}

func (m *model) updateListItemCountsFromSketch() {
	m.mu.Lock()
	for i := range m.listItems {
		item := &m.listItems[i]
		m.sketchMu.Lock()
		item.Count = m.sketch.Count(item.Item)
		m.sketchMu.Unlock()
	}
	m.mu.Unlock()
}

func (m *model) updateTopK() {
	m.sketchMu.Lock()
	items := m.sketch.SortedSlice()
	m.sketchMu.Unlock()
	m.mu.Lock()
	m.listItems = items
	m.mu.Unlock()
}

func (m *model) resizePlot(w int, h int) {
	p := plot.NewCanvas(w, h)
	p.NumDataPoints = m.plot.NumDataPoints
	p.ShowAxis = m.plot.ShowAxis
	p.LineColors = m.plot.LineColors
	m.plot = &p
}

func (m *model) updateList(msg tui.Msg) tui.Cmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]list.Item, len(m.listItems))
	order := make(map[string]int)

	m.listDelegate.Styles.SelectedTitle = m.listDelegate.Styles.SelectedTitle.Bold(m.track)
	m.listDelegate.Styles.SelectedDesc = m.listDelegate.Styles.SelectedDesc.Bold(m.track)
	m.list.SetDelegate(m.listDelegate)

	numDecimals := 1 + int(math.Ceil(math.Log10(float64(config.K+1))))
	padToItemRankWidth := strings.Repeat(" ", numDecimals+1)
	itemRankFormat := "#%-" + fmt.Sprint(numDecimals) + "d"
	for i, item := range m.listItems {
		items[i] = listItem{
			DescriptionPrefix: padToItemRankWidth,
			TitlePrefix:       fmt.Sprintf(itemRankFormat, i+1),
			Item:              item,
		}
		order[item.Item] = i
	}
	selected := m.list.SelectedItem()
	set := m.list.SetItems(items)
	var cmd tui.Cmd
	if m.track && selected != nil {
		if i, ok := order[selected.(listItem).Item.Item]; ok {
			m.list.Select(i)
		}
	}
	m.list, cmd = m.list.Update(msg)
	return tui.Batch(set, cmd)
}

func (m *model) updatePlot(_ tui.Msg) tui.Cmd {
	logScale := m.logScale.Load()

	var highlight, dim plot.Color
	if styles.DefaultRenderer().HasDarkBackground() {
		highlight, dim = plot.Cyan, plot.DimGray
	} else {
		highlight, dim = plot.Black, plot.LightGray
	}

	if len(m.listItems) == 0 {
		return nil
	}

	m.mu.Lock()
	selected := m.list.Index()
	items := m.listItems
	m.mu.Unlock()

	for i := range m.plotData {
		m.plotLineColors[i] = dim
	}
	for i := range items {
		series := m.plotData[i]
		item := items[(1+selected+i)%len(items)]

		for j := range series {
			m.sketchMu.Lock()
			value := float64(m.countAtTimeOffset(item, j))
			m.sketchMu.Unlock()
			if logScale {
				value = math.Log(max(1, value))
			}
			series[len(series)-1-j] = value
		}
	}
	n := len(items)
	m.plotLineColors[n] = highlight
	m.plotLineColors[n-1] = dim
	last := m.plotData[n]
	for j := range last {
		last[j] = 0
	}
	m.plotData[n], m.plotData[n-1] = m.plotData[n-1], m.plotData[n]
	m.mu.Lock()
	m.plotLineColors, m.plot.LineColors = m.plot.LineColors, m.plotLineColors
	m.mu.Unlock()
	m.plot.Fill(m.plotData[:n+1])
	return nil
}

func (m *model) countAtTimeOffset(item heap.Item, j int) uint32 {
	var maxCount uint32
	for k := range m.sketch.Depth {
		b := m.sketch.Buckets[topk.BucketIndex(item.Item, k, m.sketch.Width)]
		if b.Fingerprint == item.Fingerprint {
			c := uint32(b.Counts[(int(b.First)+j)%len(b.Counts)])
			maxCount = max(maxCount, c)
		}
	}
	return maxCount
}

func (m *model) View() string {
	left := m.listStyle.Render(m.list.View())
	plot := m.plot.String()

	if plot == "" {
		sb := emptyPlot(m)
		plot = sb.String()
	}

	linColor := borderFg
	logColor := borderFg
	if m.logScale.Load() {
		logColor = selectedFg
	} else {
		linColor = selectedFg
	}
	linLog := linColor.Render("LIN") + " " + logColor.Render("LOG")

	labels := ""
	if !m.latestTick.IsZero() {
		w := m.rightWidth() - 3
		leftLabel := m.latestTick.Add(-config.WindowSize).UTC().Format(time.RFC3339)
		rightLabel := m.latestTick.UTC().Format(time.RFC3339)
		space := strings.Repeat(" ", (w-len(leftLabel)-len(rightLabel)-1)/2-len("LIN LOG")/2)
		labels = " " + leftLabel + space + linLog + space + borderFg.Render(rightLabel)
	}
	right := plotStyle.Render(styles.JoinVertical(styles.Top, plot, labels))
	view := styles.JoinHorizontal(styles.Top, left, right)
	return styles.JoinVertical(styles.Left, view, m.help.View(keys))
}

func emptyPlot(m *model) strings.Builder {
	var sb strings.Builder
	if m.width < 2 || m.height < 4 {
		return sb
	}
	w, h := m.list.Width(), m.list.Height()
	sb.Grow(w * h)
	spaces := strings.Repeat(" ", m.list.Width())
	for range m.list.Height() - 2 {
		sb.WriteString(spaces)
		sb.WriteRune('\n')
	}
	return sb
}

type listItem struct {
	DescriptionPrefix string
	TitlePrefix       string
	heap.Item
}

func (i listItem) Title() string {
	return fmt.Sprintf("%s %s", i.TitlePrefix, i.Item.Item)
}
func (i listItem) Description() string {
	return fmt.Sprintf("%s %d", i.DescriptionPrefix, i.Count)
}
func (i listItem) FilterValue() string { return i.Item.Item }

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Up, k.Down, k.Track, k.Scale}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit},
		{k.Up, k.Down, k.Track, k.Scale},
	}
}

type keyMap struct {
	Track key.Binding
	Scale key.Binding
	Up    key.Binding
	Down  key.Binding
	Help  key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Track: key.NewBinding(
		key.WithKeys("t", " "),
		key.WithHelp("t/space", "track"),
	),
	Scale: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "log/lin"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
}
