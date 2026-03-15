package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Color sliding puzzle
// - Main menu with instructions
// - Adjustable grid size: 3x3, 4x4, 5x5
// - Move counter + elapsed timer
// - Victory splash animation
// - Click or arrow keys to slide tiles

const (
	windowW  = 920
	windowH  = 560
	boardTop = 80
	gap      = 5
)

var tileColors = []color.RGBA{
	{220, 60, 60, 255},
	{60, 190, 60, 255},
	{60, 90, 220, 255},
	{190, 60, 190, 255},
	{50, 205, 205, 255},
	{220, 190, 40, 255},
	{200, 200, 200, 255},
	{230, 110, 30, 255},
	{220, 80, 150, 255},
	{80, 180, 180, 255},
	{160, 100, 220, 255},
	{100, 200, 100, 255},
	{200, 150, 80, 255},
	{80, 130, 200, 255},
	{200, 200, 80, 255},
	{180, 80, 80, 255},
	{80, 200, 160, 255},
	{200, 120, 160, 255},
	{120, 160, 80, 255},
	{160, 120, 200, 255},
	{200, 160, 120, 255},
	{120, 200, 200, 255},
	{200, 200, 120, 255},
	{160, 80, 120, 255},
}

// ── Tile ──────────────────────────────────────────────────────────────────────

type Tile struct {
	Color      color.RGBA
	CorrectPos int
	IsBlank    bool
}

// ── Puzzle logic ──────────────────────────────────────────────────────────────

func createTiles(size int) []*Tile {
	n := size * size
	tiles := make([]*Tile, 0, n)
	for i := 0; i < n-1; i++ {
		tiles = append(tiles, &Tile{Color: tileColors[i%len(tileColors)], CorrectPos: i})
	}
	tiles = append(tiles, &Tile{IsBlank: true, CorrectPos: n - 1})
	return tiles
}

func createBoard(tiles []*Tile, size int) [][]*Tile {
	b := make([][]*Tile, size)
	for i := range b {
		b[i] = make([]*Tile, size)
		for j := range b[i] {
			b[i][j] = tiles[i*size+j]
		}
	}
	return b
}

func cloneBoard(src [][]*Tile) [][]*Tile {
	dst := make([][]*Tile, len(src))
	for i := range src {
		dst[i] = make([]*Tile, len(src[i]))
		copy(dst[i], src[i])
	}
	return dst
}

func countInversions(tiles []*Tile) int {
	n := 0
	for i := 0; i < len(tiles); i++ {
		if tiles[i].IsBlank {
			continue
		}
		for j := i + 1; j < len(tiles); j++ {
			if tiles[j].IsBlank {
				continue
			}
			if tiles[i].CorrectPos > tiles[j].CorrectPos {
				n++
			}
		}
	}
	return n
}

func isSolvable(tiles []*Tile, size int) bool {
	inv := countInversions(tiles)
	if size%2 == 1 {
		return inv%2 == 0
	}
	blankRow := 0
	for i, t := range tiles {
		if t.IsBlank {
			blankRow = size - i/size
			break
		}
	}
	return (inv+blankRow)%2 == 1
}

func shuffleSolvable(tiles []*Tile, size int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		for i := len(tiles) - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			tiles[i], tiles[j] = tiles[j], tiles[i]
		}
		if isSolvable(tiles, size) {
			break
		}
	}
}

func findBlank(board [][]*Tile) (int, int) {
	for i := range board {
		for j := range board[i] {
			if board[i][j].IsBlank {
				return i, j
			}
		}
	}
	return -1, -1
}

func isAdjacent(board [][]*Tile, row, col int) bool {
	br, bc := findBlank(board)
	dr, dc := row-br, col-bc
	return (dr == 0 && (dc == 1 || dc == -1)) || (dc == 0 && (dr == 1 || dr == -1))
}

func trySlide(board [][]*Tile, row, col int) bool {
	if !isAdjacent(board, row, col) {
		return false
	}
	br, bc := findBlank(board)
	board[br][bc], board[row][col] = board[row][col], board[br][bc]
	return true
}

func checkWin(current, goal [][]*Tile) bool {
	for i := range current {
		for j := range current[i] {
			if current[i][j] != goal[i][j] {
				return false
			}
		}
	}
	return true
}

func shuffleBoard(board [][]*Tile, moves int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + 1))
	size := len(board)
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for i := 0; i < moves; i++ {
		br, bc := findBlank(board)
		d := dirs[rng.Intn(4)]
		nr, nc := br+d[0], bc+d[1]
		if nr >= 0 && nr < size && nc >= 0 && nc < size {
			board[br][bc], board[nr][nc] = board[nr][nc], board[br][bc]
		}
	}
}

// ── Layout helpers ────────────────────────────────────────────────────────────

func tileSize(size int) int {
	switch size {
	case 3:
		return 110
	case 4:
		return 88
	case 5:
		return 72
	}
	return 88
}

func boardWidth(size int) int {
	ts := tileSize(size)
	return size*ts + (size-1)*gap
}

func leftBoardOriginX(size int) int {
	bw := boardWidth(size)
	totalW := bw*2 + 80
	return (windowW - totalW) / 2
}

func rightBoardOriginX(size int) int {
	return leftBoardOriginX(size) + boardWidth(size) + 80
}

func pixelToTile(px, py, originX, size int) (int, int) {
	ts := tileSize(size)
	lx := px - originX
	ly := py - boardTop
	if lx < 0 || ly < 0 {
		return -1, -1
	}
	col := lx / (ts + gap)
	row := ly / (ts + gap)
	if col >= size || row >= size {
		return -1, -1
	}
	if lx-col*(ts+gap) >= ts || ly-row*(ts+gap) >= ts {
		return -1, -1
	}
	return row, col
}

// ── Game state ────────────────────────────────────────────────────────────────

type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StateWon
)

type VictoryParticle struct {
	x, y    float32
	vx, vy  float32
	life    int
	maxLife int
	size    float32
	color   color.RGBA
}

type Game struct {
	size    int
	goal    [][]*Tile
	current [][]*Tile
	state   GameState

	moves     int
	startTime time.Time
	elapsed   time.Duration

	// Victory animation
	victoryT  int
	particles []VictoryParticle

	// Menu — selected grid size
	menuSize int

	rng *rand.Rand
}

func newGame(size int) *Game {
	tiles := createTiles(size)
	shuffleSolvable(tiles, size)
	goal := createBoard(tiles, size)
	current := cloneBoard(goal)
	shuffleBoard(current, 80+size*20)
	for checkWin(current, goal) {
		shuffleBoard(current, 30)
	}
	return &Game{
		size:      size,
		goal:      goal,
		current:   current,
		startTime: time.Now(),
		state:     StatePlaying,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func initialGame() *Game {
	return &Game{
		state:    StateMenu,
		menuSize: 3,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (g *Game) Update() error {
	switch g.state {
	case StateMenu:
		g.updateMenu()
	case StatePlaying:
		g.updatePlaying()
	case StateWon:
		g.updateWon()
	}
	return nil
}

func (g *Game) updateMenu() {
	// Select grid size with number keys
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.menuSize = 3
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.menuSize = 4
	}
	if inpututil.IsKeyJustPressed(ebiten.Key5) {
		g.menuSize = 5
	}
	// Start game
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		*g = *newGame(g.menuSize)
	}
}

func (g *Game) updatePlaying() {
	// Return to menu
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateMenu
		g.menuSize = g.size
		return
	}

	// New game (same size)
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		*g = *newGame(g.size)
		return
	}

	// Elapsed timer
	g.elapsed = time.Since(g.startTime)

	// Mouse click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		row, col := pixelToTile(mx, my, rightBoardOriginX(g.size), g.size)
		if row >= 0 && col >= 0 {
			g.doSlide(row, col)
		}
	}

}

func (g *Game) updateWon() {
	g.victoryT++
	for i := range g.particles {
		p := &g.particles[i]
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.12
		p.vx *= 0.98
		p.life--
	}
	// New game or menu
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		*g = *newGame(g.size)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateMenu
		g.menuSize = g.size
	}
}

func (g *Game) doSlide(row, col int) {
	if trySlide(g.current, row, col) {
		g.moves++
		if checkWin(g.current, g.goal) {
			g.state = StateWon
			g.elapsed = time.Since(g.startTime)
			g.spawnVictoryParticles()
		}
	}
}

func (g *Game) spawnVictoryParticles() {
	rx := rightBoardOriginX(g.size)
	ts := tileSize(g.size)
	bw := boardWidth(g.size)
	bh := g.size*(ts+gap) - gap
	cx := float32(rx) + float32(bw)/2
	cy := float32(boardTop) + float32(bh)/2

	splashColors := []color.RGBA{
		{255, 220, 50, 255},
		{255, 100, 80, 255},
		{100, 200, 255, 255},
		{180, 100, 255, 255},
		{80, 230, 120, 255},
		{255, 160, 40, 255},
	}
	for i := 0; i < 80; i++ {
		angle := g.rng.Float64() * math.Pi * 2
		speed := float32(2.5 + g.rng.Float64()*5.5)
		life := 40 + g.rng.Intn(40)
		c := splashColors[g.rng.Intn(len(splashColors))]
		g.particles = append(g.particles, VictoryParticle{
			x:       cx + float32(g.rng.Float64()-0.5)*float32(bw)*0.3,
			y:       cy + float32(g.rng.Float64()-0.5)*float32(bh)*0.3,
			vx:      float32(math.Cos(angle)) * speed,
			vy:      float32(math.Sin(angle)) * speed * 0.7,
			life:    life,
			maxLife: life,
			size:    float32(3 + g.rng.Intn(5)),
			color:   c,
		})
	}
}

// ── Draw ──────────────────────────────────────────────────────────────────────

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case StateMenu:
		g.drawMenu(screen)
	case StatePlaying:
		g.drawGame(screen)
	case StateWon:
		g.drawGame(screen)
		g.drawWin(screen)
	}
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 18, 28, 255})

	// Draw some decorative tiles across the top
	demoColors := tileColors[:8]
	for i, c := range demoColors {
		x := float32(windowW/2-4*50+i*50) - 25
		vector.DrawFilledRect(screen, x, 30, 44, 44, c, false)
		light := lighten(c, 60)
		dark := darken(c, 50)
		vector.DrawFilledRect(screen, x, 30, 44, 4, light, false)
		vector.DrawFilledRect(screen, x, 30, 4, 44, light, false)
		vector.DrawFilledRect(screen, x, 70, 44, 4, dark, false)
		vector.DrawFilledRect(screen, x+40, 30, 4, 44, dark, false)
	}

	// Title
	ebitenutil.DebugPrintAt(screen, "COLOR  SLIDING  PUZZLE", windowW/2-80, 100)

	// Divider line
	vector.StrokeLine(screen, float32(windowW/2-180), 120, float32(windowW/2+180), 120, 1, color.RGBA{80, 80, 100, 255}, false)

	// Instructions
	lines := []string{
		"HOW TO PLAY",
		"",
		"The goal is to arrange the puzzle tiles",
		"so they match the GOAL board on the left.",
		"",
		"Click a tile next to the blank space to slide it.",
		"",
		"CONTROLS",
		"",
		"Click tile  —  slide it into the blank space",
		"N                   —  new game (same size)",
		"ESC                 —  return to this menu",
		"",
		"GRID SIZE",
		"",
		"3  —  3x3  (easy)",
		"4  —  4x4  (medium)",
		"5  —  5x5  (hard)",
	}

	for i, line := range lines {
		c := color.RGBA{200, 200, 220, 255}
		if line == "HOW TO PLAY" || line == "CONTROLS" || line == "GRID SIZE" {
			c = color.RGBA{255, 220, 80, 255}
		}
		_ = c
		ebitenutil.DebugPrintAt(screen, line, windowW/2-160, 138+i*16)
	}

	// Grid size selector
	sizes := []struct {
		n     int
		label string
	}{{3, "3x3"}, {4, "4x4"}, {5, "5x5"}}

	for i, s := range sizes {
		bx := float32(windowW/2 - 130 + i*100)
		by := float32(450)
		bw := float32(80)
		bh := float32(30)
		bg := color.RGBA{40, 40, 60, 255}
		if g.menuSize == s.n {
			bg = color.RGBA{80, 120, 220, 255}
		}
		vector.DrawFilledRect(screen, bx, by, bw, bh, bg, false)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("[%d] %s", s.n, s.label), int(bx)+16, int(by)+10)
	}

	// Start prompt
	ebitenutil.DebugPrintAt(screen, "Press SPACE or ENTER to start", windowW/2-100, 500)
}

func (g *Game) drawGame(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 18, 28, 255})

	lx := leftBoardOriginX(g.size)
	rx := rightBoardOriginX(g.size)
	ts := tileSize(g.size)

	mx, my := ebiten.CursorPosition()
	hoverRow, hoverCol := -1, -1
	if g.state == StatePlaying {
		hoverRow, hoverCol = pixelToTile(mx, my, rx, g.size)
	}

	// Draw boards
	g.drawBoard(screen, g.goal, lx, ts, -1, -1)
	g.drawBoard(screen, g.current, rx, ts, hoverRow, hoverCol)

	// Board labels
	bw := float32(boardWidth(g.size))
	ebitenutil.DebugPrintAt(screen, "G O A L", lx+int(bw/2)-24, boardTop-44)
	ebitenutil.DebugPrintAt(screen, "P U Z Z L E", rx+int(bw/2)-36, boardTop-44)

	// Stats (below puzzle board)
	elapsed := g.elapsed
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	statsY := boardTop + g.size*(ts+gap) + 14
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Time:  %02d:%02d", mins, secs), rx, statsY)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Moves: %d", g.moves), rx, statsY+16)

	// Minimal footer
	ebitenutil.DebugPrintAt(screen, "N = new game   ESC = menu", lx, windowH-20)
}

func (g *Game) drawWin(screen *ebiten.Image) {
	elapsed := g.elapsed
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60

	// Win message at top
	msg := fmt.Sprintf("SOLVED!   %d moves   %02d:%02d", g.moves, mins, secs)
	ebitenutil.DebugPrintAt(screen, msg, windowW/2-len(msg)*3, boardTop-58)
	ebitenutil.DebugPrintAt(screen, "N = new game   ESC = menu", windowW/2-80, boardTop-42)

	// Splash particles
	for _, p := range g.particles {
		if p.life <= 0 {
			continue
		}
		alpha := float32(p.life) / float32(p.maxLife)
		c := color.RGBA{p.color.R, p.color.G, p.color.B, uint8(alpha * 255)}
		vector.DrawFilledCircle(screen, p.x, p.y, p.size*alpha, c, false)
	}
}

func (g *Game) drawBoard(screen *ebiten.Image, board [][]*Tile, boardX, ts, hoverRow, hoverCol int) {
	for row := 0; row < g.size; row++ {
		for col := 0; col < g.size; col++ {
			tile := board[row][col]
			x := float32(boardX + col*(ts+gap))
			y := float32(boardTop + row*(ts+gap))
			tsf := float32(ts)

			if tile.IsBlank {
				vector.DrawFilledRect(screen, x, y, tsf, tsf, color.RGBA{35, 35, 45, 255}, false)
				continue
			}

			// Hover highlight
			if row == hoverRow && col == hoverCol && isAdjacent(board, row, col) {
				vector.DrawFilledRect(screen, x-3, y-3, tsf+6, tsf+6, color.RGBA{255, 255, 255, 200}, false)
			}

			tileColor := tile.Color
			vector.DrawFilledRect(screen, x, y, tsf, tsf, tileColor, false)

			dark := darken(tileColor, 50)
			light := lighten(tileColor, 60)
			vector.DrawFilledRect(screen, x, y+tsf-4, tsf, 4, dark, false)
			vector.DrawFilledRect(screen, x+tsf-4, y, 4, tsf, dark, false)
			vector.DrawFilledRect(screen, x, y, tsf, 4, light, false)
			vector.DrawFilledRect(screen, x, y, 4, tsf, light, false)
		}
	}
}

func (g *Game) Layout(_, _ int) (int, int) {
	return windowW, windowH
}

// ── Color helpers ─────────────────────────────────────────────────────────────

func darken(c color.RGBA, d uint8) color.RGBA {
	sub := func(a, b uint8) uint8 {
		if a < b {
			return 0
		}
		return a - b
	}
	return color.RGBA{sub(c.R, d), sub(c.G, d), sub(c.B, d), c.A}
}

func lighten(c color.RGBA, d uint8) color.RGBA {
	add := func(a, b uint8) uint8 {
		if int(a)+int(b) > 255 {
			return 255
		}
		return a + b
	}
	return color.RGBA{add(c.R, d), add(c.G, d), add(c.B, d), c.A}
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowTitle("Color Sliding Puzzle")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	if err := ebiten.RunGame(initialGame()); err != nil {
		log.Fatal(err)
	}
}
