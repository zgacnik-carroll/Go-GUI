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

// Color sliding puzzle — improved version.
// - Adjustable grid size: 3×3, 4×4, 5×5
// - Move counter + elapsed timer
// - Victory animation (tiles pulse and flash)
// - Click or arrow keys to slide tiles
// - N = new game, 3/4/5 = change grid size

const (
	windowW  = 920
	windowH  = 560
	boardTop = 80 // y pixel where boards start
	gap      = 5
)

// tileColors has enough entries for a 5×5 grid (24 tiles).
var tileColors = []color.RGBA{
	{220, 60, 60, 255},   // Red
	{60, 190, 60, 255},   // Green
	{60, 90, 220, 255},   // Blue
	{190, 60, 190, 255},  // Magenta
	{50, 205, 205, 255},  // Cyan
	{220, 190, 40, 255},  // Yellow
	{200, 200, 200, 255}, // Light gray
	{230, 110, 30, 255},  // Orange
	{220, 80, 150, 255},  // Pink
	{80, 180, 180, 255},  // Teal
	{160, 100, 220, 255}, // Violet
	{100, 200, 100, 255}, // Lime
	{200, 150, 80, 255},  // Tan
	{80, 130, 200, 255},  // Sky blue
	{200, 200, 80, 255},  // Khaki
	{180, 80, 80, 255},   // Brick
	{80, 200, 160, 255},  // Mint
	{200, 120, 160, 255}, // Rose
	{120, 160, 80, 255},  // Olive
	{160, 120, 200, 255}, // Lavender
	{200, 160, 120, 255}, // Peach
	{120, 200, 200, 255}, // Aqua
	{200, 200, 120, 255}, // Cream
	{160, 80, 120, 255},  // Maroon
}

// ── Tile ──────────────────────────────────────────────────────────────────────

type Tile struct {
	Color      color.RGBA
	CorrectPos int
	IsBlank    bool
}

// ── Puzzle logic (grid-size agnostic) ────────────────────────────────────────

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

// isSolvable returns true if the shuffled tile slice represents a solvable puzzle.
// For odd-width grids: solvable iff inversions are even.
// For even-width grids: solvable iff (inversions + blank row from bottom) is odd.
func isSolvable(tiles []*Tile, size int) bool {
	inv := countInversions(tiles)
	if size%2 == 1 {
		return inv%2 == 0
	}
	// Find blank row from bottom (1-indexed)
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

// tileSize computes tile size based on grid size to keep boards fitting.
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

// leftBoardX and rightBoardX computed dynamically.
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
	StatePlaying GameState = iota
	StateWon
)

// VictoryParticle is one particle in the splash burst.
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
	elapsed   time.Duration // frozen on win

	// Victory animation
	victoryT  int
	particles []VictoryParticle

	// Best scores per grid size
	bestMoves [6]int // index = size
	bestTime  [6]time.Duration

	rng *rand.Rand
}

func newGame(size int, best *Game) *Game {
	tiles := createTiles(size)
	shuffleSolvable(tiles, size)
	goal := createBoard(tiles, size)
	current := cloneBoard(goal)
	shuffleBoard(current, 80+size*20)
	for checkWin(current, goal) {
		shuffleBoard(current, 30)
	}
	g := &Game{
		size:      size,
		goal:      goal,
		current:   current,
		startTime: time.Now(),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if best != nil {
		g.bestMoves = best.bestMoves
		g.bestTime = best.bestTime
	}
	return g
}

func (g *Game) recordBest() {
	sz := g.size
	if g.bestMoves[sz] == 0 || g.moves < g.bestMoves[sz] {
		g.bestMoves[sz] = g.moves
	}
	if g.bestTime[sz] == 0 || g.elapsed < g.bestTime[sz] {
		g.bestTime[sz] = g.elapsed
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (g *Game) Update() error {
	// Grid size hotkeys
	for _, kv := range []struct {
		key ebiten.Key
		sz  int
	}{
		{ebiten.Key3, 3}, {ebiten.Key4, 4}, {ebiten.Key5, 5},
	} {
		if inpututil.IsKeyJustPressed(kv.key) && g.size != kv.sz {
			*g = *newGame(kv.sz, g)
			return nil
		}
	}

	// New game
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		*g = *newGame(g.size, g)
		return nil
	}

	// Victory animation ticks
	if g.state == StateWon {
		g.victoryT++
		for i := range g.particles {
			p := &g.particles[i]
			p.x += p.vx
			p.y += p.vy
			p.vy += 0.12 // gravity
			p.vx *= 0.98 // drag
			p.life--
		}
		return nil
	}

	// Elapsed timer
	g.elapsed = time.Since(g.startTime)

	// ── Mouse click ───────────────────────────────────────────────────────
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		row, col := pixelToTile(mx, my, rightBoardOriginX(g.size), g.size)
		if row >= 0 && col >= 0 {
			g.doSlide(row, col)
		}
	}

	// ── Arrow keys ────────────────────────────────────────────────────────
	// Arrow keys move the blank in the given direction
	// (i.e. slide the tile on the opposite side into the blank)
	br, bc := findBlank(g.current)
	arrowMoves := map[ebiten.Key][2]int{
		ebiten.KeyArrowUp:    {br + 1, bc},
		ebiten.KeyArrowDown:  {br - 1, bc},
		ebiten.KeyArrowLeft:  {br, bc + 1},
		ebiten.KeyArrowRight: {br, bc - 1},
	}
	for key, rc := range arrowMoves {
		if inpututil.IsKeyJustPressed(key) {
			r, c := rc[0], rc[1]
			if r >= 0 && r < g.size && c >= 0 && c < g.size {
				g.doSlide(r, c)
			}
		}
	}

	return nil
}

func (g *Game) doSlide(row, col int) {
	if trySlide(g.current, row, col) {
		g.moves++
		if checkWin(g.current, g.goal) {
			g.state = StateWon
			g.elapsed = time.Since(g.startTime)
			g.recordBest()
			g.spawnVictoryParticles()
		}
	}
}

func (g *Game) spawnVictoryParticles() {
	rx := rightBoardOriginX(g.size)
	ts := tileSize(g.size)
	bw := boardWidth(g.size)
	bh := g.size*(ts+gap) - gap

	// Center of the puzzle board
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
	screen.Fill(color.RGBA{18, 18, 28, 255})

	lx := leftBoardOriginX(g.size)
	rx := rightBoardOriginX(g.size)
	ts := tileSize(g.size)

	mx, my := ebiten.CursorPosition()
	hoverRow, hoverCol := pixelToTile(mx, my, rx, g.size)

	// Goal board
	g.drawBoard(screen, g.goal, lx, ts, -1, -1)
	// Puzzle board
	g.drawBoard(screen, g.current, rx, ts, hoverRow, hoverCol)

	// Labels
	bw := float32(boardWidth(g.size))
	ebitenutil.DebugPrintAt(screen, "G O A L", lx+int(bw/2)-24, boardTop-44)
	ebitenutil.DebugPrintAt(screen, "P U Z Z L E", rx+int(bw/2)-36, boardTop-44)

	// Timer
	elapsed := g.elapsed
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Time: %02d:%02d", mins, secs), rx, boardTop+g.size*(ts+gap)+14)

	// Moves
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Moves: %d", g.moves), rx, boardTop+g.size*(ts+gap)+30)

	// Best scores for this grid size
	sz := g.size
	if g.bestMoves[sz] > 0 {
		bt := g.bestTime[sz]
		bm := int(bt.Minutes())
		bs := int(bt.Seconds()) % 60
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Best: %d moves, %02d:%02d", g.bestMoves[sz], bm, bs), rx, boardTop+g.size*(ts+gap)+46)
	}

	// Grid size selector
	ebitenutil.DebugPrintAt(screen, "Grid: [3] 3×3   [4] 4×4   [5] 5×5", lx, windowH-38)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Current: %d×%d   N = New Game   ↑↓←→ or click to slide", g.size, g.size), lx, windowH-22)

	// Win overlay
	if g.state == StateWon {
		msg := fmt.Sprintf("✓ SOLVED!  %d moves  |  %02d:%02d", g.moves, mins, secs)
		ebitenutil.DebugPrintAt(screen, msg, lx, boardTop-62)
		ebitenutil.DebugPrintAt(screen, "Press N for a new game", lx, boardTop-46)
		g.drawSplash(screen)
	}
}

// drawBoard renders a size×size board with optional hover highlight and victory animation.
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

			// Tile body
			tileColor := tile.Color
			vector.DrawFilledRect(screen, x, y, tsf, tsf, tileColor, false)

			// Bevel
			dark := darken(tileColor, 50)
			light := lighten(tileColor, 60)
			vector.DrawFilledRect(screen, x, y+tsf-4, tsf, 4, dark, false)
			vector.DrawFilledRect(screen, x+tsf-4, y, 4, tsf, dark, false)
			vector.DrawFilledRect(screen, x, y, tsf, 4, light, false)
			vector.DrawFilledRect(screen, x, y, 4, tsf, light, false)
		}
	}
}

// drawSplash renders the victory burst particles.
func (g *Game) drawSplash(screen *ebiten.Image) {
	for _, p := range g.particles {
		if p.life <= 0 {
			continue
		}
		alpha := float32(p.life) / float32(p.maxLife)
		c := color.RGBA{p.color.R, p.color.G, p.color.B, uint8(alpha * 255)}
		vector.DrawFilledCircle(screen, p.x, p.y, p.size*alpha, c, false)
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

	if err := ebiten.RunGame(newGame(3, nil)); err != nil {
		log.Fatal(err)
	}
}
