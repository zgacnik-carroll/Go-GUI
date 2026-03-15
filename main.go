// This program implements a color-based sliding puzzle game using the Ebitengine
// 2D game library. The game presents two boards side by side: a goal board showing
// the solved arrangement, and a puzzle board the player must manipulate to match it.
//
// Players click tiles adjacent to the blank space to slide them into position.
// Three grid sizes are supported (3×3, 4×4, 5×5), selectable from the main menu.
// A move counter and elapsed timer track performance, and a particle splash
// animation plays upon solving the puzzle.
//
// Controls:
//   - Click a tile adjacent to the blank — slide it
//   - N — start a new game (same grid size)
//   - ESC — return to the main menu
//   - 3 / 4 / 5 — select grid size on the menu
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

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	// windowW is the fixed width of the game window in pixels.
	windowW = 920
	// windowH is the fixed height of the game window in pixels.
	windowH = 560
	// boardTop is the y-coordinate (pixels from the top) where both boards begin.
	boardTop = 80
	// gap is the number of pixels between adjacent tiles on a board.
	gap = 5
)

// tileColors defines the distinct background colors assigned to each puzzle tile.
// The slice must contain at least (size*size - 1) entries for the largest supported
// grid (5×5 = 24 colored tiles). Colors are assigned in order by tile index.
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

// Tile represents a single cell in the sliding puzzle.
// Tiles are shared by pointer between the goal board and the current board,
// meaning each unique tile exists only once in memory. The current board stores
// a different arrangement of the same pointers as the goal board.
type Tile struct {
	// Color is the background color drawn for this tile.
	Color color.RGBA
	// CorrectPos is the zero-based index of this tile's position in the solved board,
	// reading left-to-right, top-to-bottom. Used to verify the win condition.
	CorrectPos int
	// IsBlank marks the one empty cell that tiles slide into. The blank tile is
	// always the last tile created (index size*size-1).
	IsBlank bool
}

// ── Puzzle logic ──────────────────────────────────────────────────────────────

// createTiles builds the initial ordered set of tiles for a size×size puzzle.
// It creates (size*size - 1) colored tiles followed by one blank tile.
// Tiles are assigned colors from tileColors in order, wrapping if necessary.
// CorrectPos is set to the tile's creation index so the solved order is 0, 1, 2, …
func createTiles(size int) []*Tile {
	n := size * size
	tiles := make([]*Tile, 0, n)
	for i := 0; i < n-1; i++ {
		tiles = append(tiles, &Tile{Color: tileColors[i%len(tileColors)], CorrectPos: i})
	}
	// The blank tile occupies the last position in the solved arrangement.
	tiles = append(tiles, &Tile{IsBlank: true, CorrectPos: n - 1})
	return tiles
}

// createBoard converts a flat slice of size*size tiles into a 2D size×size board.
// Tiles are placed row-by-row, left-to-right. The returned board is a slice of
// row slices, so board[row][col] accesses the tile at that position.
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

// cloneBoard creates a shallow copy of a board, duplicating the row and column
// slices but sharing the underlying Tile pointers. This means both boards refer
// to the same Tile objects — only their arrangement differs. Tiles themselves
// are never mutated after creation, so sharing is safe.
func cloneBoard(src [][]*Tile) [][]*Tile {
	dst := make([][]*Tile, len(src))
	for i := range src {
		dst[i] = make([]*Tile, len(src[i]))
		copy(dst[i], src[i])
	}
	return dst
}

// countInversions counts the number of inversions in a flat tile slice.
// An inversion is a pair (i, j) where i < j but tiles[i].CorrectPos > tiles[j].CorrectPos,
// meaning tile i should appear after tile j in the solved order. The blank tile
// is ignored. Inversion count is used to determine whether a shuffle is solvable.
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

// isSolvable reports whether a shuffled tile arrangement can be solved by legal
// sliding moves. The rule differs based on grid width:
//
//   - Odd-width grid (3×3, 5×5): solvable if and only if the inversion count is even.
//   - Even-width grid (4×4): solvable if and only if (inversions + blank row from bottom) is odd.
//
// These rules follow from the mathematical theory of permutations and the
// structure of the 15-puzzle and its generalizations.
func isSolvable(tiles []*Tile, size int) bool {
	inv := countInversions(tiles)
	if size%2 == 1 {
		// Odd grid: parity of inversions alone determines solvability.
		return inv%2 == 0
	}
	// Even grid: also factor in the blank tile's row position (counted from the bottom, 1-indexed).
	blankRow := 0
	for i, t := range tiles {
		if t.IsBlank {
			blankRow = size - i/size
			break
		}
	}
	return (inv+blankRow)%2 == 1
}

// shuffleSolvable randomly permutes tiles using a Fisher-Yates shuffle,
// retrying until the resulting arrangement satisfies isSolvable. In practice
// roughly half of all shuffles are solvable, so this loop terminates quickly.
// The shuffle uses a time-seeded RNG so each call produces a different result.
func shuffleSolvable(tiles []*Tile, size int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		// Fisher-Yates: iterate backwards, swapping each element with a random
		// earlier (or equal) element to produce a uniformly random permutation.
		for i := len(tiles) - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			tiles[i], tiles[j] = tiles[j], tiles[i]
		}
		if isSolvable(tiles, size) {
			break
		}
	}
}

// findBlank searches the board and returns the (row, col) coordinates of the
// blank tile. Returns (-1, -1) if no blank tile is found, which should never
// happen in a well-formed board.
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

// isAdjacent reports whether the tile at (row, col) is directly adjacent
// (up, down, left, or right — not diagonal) to the blank tile on the board.
// Only adjacent tiles are legal moves in the sliding puzzle.
func isAdjacent(board [][]*Tile, row, col int) bool {
	br, bc := findBlank(board)
	dr, dc := row-br, col-bc
	return (dr == 0 && (dc == 1 || dc == -1)) || (dc == 0 && (dr == 1 || dr == -1))
}

// trySlide attempts to slide the tile at (row, col) into the blank space.
// The move is only legal if the tile is adjacent to the blank. If legal,
// the tile and blank are swapped in place and true is returned. Returns false
// if the move is not legal and the board is left unchanged.
func trySlide(board [][]*Tile, row, col int) bool {
	if !isAdjacent(board, row, col) {
		return false
	}
	br, bc := findBlank(board)
	board[br][bc], board[row][col] = board[row][col], board[br][bc]
	return true
}

// checkWin reports whether the current board exactly matches the goal board.
// Since both boards share the same Tile pointers, equality is checked by
// pointer comparison (current[i][j] == goal[i][j]) rather than by value.
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

// shuffleBoard scrambles a board by performing a number of random legal moves
// starting from the current state. This guarantees the result is always solvable
// (unlike a random permutation) because every move is reversible. A separate
// time-seeded RNG (+1 offset) is used so it differs from shuffleSolvable's RNG.
func shuffleBoard(board [][]*Tile, moves int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + 1))
	size := len(board)
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for i := 0; i < moves; i++ {
		br, bc := findBlank(board)
		d := dirs[rng.Intn(4)]
		nr, nc := br+d[0], bc+d[1]
		// Only perform the move if the target cell is within bounds.
		if nr >= 0 && nr < size && nc >= 0 && nc < size {
			board[br][bc], board[nr][nc] = board[nr][nc], board[br][bc]
		}
	}
}

// ── Layout helpers ────────────────────────────────────────────────────────────

// tileSize returns the pixel width (and height) of a single tile for the given
// grid size. Larger grids use smaller tiles so both boards fit within windowW.
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

// boardWidth returns the total pixel width occupied by a size×size board,
// including the tiles and all inter-tile gaps.
func boardWidth(size int) int {
	ts := tileSize(size)
	return size*ts + (size-1)*gap
}

// leftBoardOriginX returns the x pixel coordinate of the left edge of the
// goal (left) board, calculated so both boards are centered horizontally
// within the window with an 80-pixel gap between them.
func leftBoardOriginX(size int) int {
	bw := boardWidth(size)
	totalW := bw*2 + 80
	return (windowW - totalW) / 2
}

// rightBoardOriginX returns the x pixel coordinate of the left edge of the
// puzzle (right) board. It is positioned immediately after the goal board
// plus the 80-pixel inter-board gap.
func rightBoardOriginX(size int) int {
	return leftBoardOriginX(size) + boardWidth(size) + 80
}

// pixelToTile converts a screen pixel coordinate (px, py) to a (row, col) tile
// index for a board whose left edge starts at originX. Returns (-1, -1) if the
// coordinate falls outside the board or within the gap between tiles.
// The boardTop constant is subtracted from py to account for the vertical offset.
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
	// Reject clicks that land in the gap pixels between tiles.
	if lx-col*(ts+gap) >= ts || ly-row*(ts+gap) >= ts {
		return -1, -1
	}
	return row, col
}

// ── Game state ────────────────────────────────────────────────────────────────

// GameState represents the current high-level phase of the application.
type GameState int

const (
	// StateMenu is the main menu screen where the player chooses a grid size.
	StateMenu GameState = iota
	// StatePlaying is the active puzzle-solving phase.
	StatePlaying
	// StateWon is displayed after the player successfully solves the puzzle.
	StateWon
)

// VictoryParticle is one particle in the confetti burst played when the puzzle
// is solved. Each particle has a position, velocity, remaining lifetime, size,
// and color. Gravity is applied each tick and velocity decays slightly (drag).
type VictoryParticle struct {
	x, y    float32 // current screen position
	vx, vy  float32 // velocity in pixels per tick
	life    int     // remaining ticks before the particle disappears
	maxLife int     // total lifetime used to compute alpha fade
	size    float32 // radius in pixels
	color   color.RGBA
}

// Game holds all mutable state for the application and implements ebiten.Game.
type Game struct {
	// size is the current grid dimension (3, 4, or 5).
	size int
	// goal is the solved board that the player is trying to recreate.
	goal [][]*Tile
	// current is the board the player is actively manipulating.
	current [][]*Tile
	// state is the current phase of the game (menu, playing, or won).
	state GameState

	// moves is the number of successful tile slides the player has made.
	moves int
	// startTime records when the current game began, used to compute elapsed.
	startTime time.Time
	// elapsed holds the final solve duration, frozen at the moment of winning.
	// While playing, elapsed is recomputed each tick from startTime.
	elapsed time.Duration

	// victoryT counts ticks since the win state was entered; used by the
	// particle animation to drive physics updates.
	victoryT int
	// particles holds the active confetti particles for the victory animation.
	particles []VictoryParticle

	// menuSize tracks which grid size is currently highlighted in the menu.
	menuSize int

	// rng is a seeded random number generator used for shuffling and particles.
	rng *rand.Rand
}

// newGame creates and returns a fully initialised Game ready to play at the
// given grid size. It generates a random solvable goal layout, clones it as
// the starting current board, then scrambles the current board with legal moves.
// If the scramble accidentally produces the solved state it re-scrambles.
func newGame(size int) *Game {
	tiles := createTiles(size)
	shuffleSolvable(tiles, size)
	goal := createBoard(tiles, size)

	// Clone the goal layout so current and goal share Tile pointers
	// but have independent row/column slices.
	current := cloneBoard(goal)
	shuffleBoard(current, 80+size*20) // more moves for larger grids = harder start
	for checkWin(current, goal) {
		// Re-scramble in the rare case shuffleBoard produces the solved state.
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

// initialGame returns a Game in the StateMenu phase with no puzzle loaded.
// This is the entry point used by main — the player picks a grid size and
// presses SPACE/ENTER before a puzzle is generated.
func initialGame() *Game {
	return &Game{
		state:    StateMenu,
		menuSize: 3, // default selection: 3×3 (easiest)
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

// Update is called by Ebitengine once per tick (60 times per second by default).
// It dispatches to the appropriate handler based on the current game state.
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

// updateMenu handles input while the main menu is displayed.
// Number keys 3–5 change the highlighted grid size. SPACE or ENTER
// starts a new game with the currently selected size.
func (g *Game) updateMenu() {
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.menuSize = 3
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		g.menuSize = 4
	}
	if inpututil.IsKeyJustPressed(ebiten.Key5) {
		g.menuSize = 5
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		*g = *newGame(g.menuSize)
	}
}

// updatePlaying handles input and state during active gameplay.
// It updates the elapsed timer each tick, processes mouse clicks on the puzzle
// board, and checks for the ESC and N hotkeys.
func (g *Game) updatePlaying() {
	// ESC returns to the menu, preserving the selected grid size.
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateMenu
		g.menuSize = g.size
		return
	}

	// N starts a fresh game at the same grid size without returning to the menu.
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		*g = *newGame(g.size)
		return
	}

	// Recompute elapsed every tick so the timer display stays current.
	g.elapsed = time.Since(g.startTime)

	// Convert a left-click to a tile coordinate on the puzzle board and attempt
	// to slide that tile. Clicks on the goal board or empty space are ignored.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		row, col := pixelToTile(mx, my, rightBoardOriginX(g.size), g.size)
		if row >= 0 && col >= 0 {
			g.doSlide(row, col)
		}
	}
}

// updateWon advances the victory particle animation each tick and listens for
// N (new game) and ESC (return to menu) while the win screen is shown.
func (g *Game) updateWon() {
	g.victoryT++

	// Advance every particle: apply velocity, gravity, and drag, then age it.
	for i := range g.particles {
		p := &g.particles[i]
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.12 // gravity pulls particles downward each tick
		p.vx *= 0.98 // light horizontal drag gradually slows particles
		p.life--
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		*g = *newGame(g.size)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateMenu
		g.menuSize = g.size
	}
}

// doSlide attempts to slide the tile at (row, col) on the current board.
// If the slide is legal, the move counter is incremented. If the resulting
// board matches the goal the game transitions to StateWon, the elapsed time
// is frozen, and the victory particle burst is spawned.
func (g *Game) doSlide(row, col int) {
	if trySlide(g.current, row, col) {
		g.moves++
		if checkWin(g.current, g.goal) {
			g.state = StateWon
			g.elapsed = time.Since(g.startTime) // freeze the timer
			g.spawnVictoryParticles()
		}
	}
}

// spawnVictoryParticles fills g.particles with 80 confetti particles that burst
// outward from the center of the puzzle board. Each particle is given a random
// angle, speed, lifetime, size, and color. The spawn area is slightly randomised
// around the board center so the burst looks organic rather than point-like.
func (g *Game) spawnVictoryParticles() {
	rx := rightBoardOriginX(g.size)
	ts := tileSize(g.size)
	bw := boardWidth(g.size)
	bh := g.size*(ts+gap) - gap

	// Compute the screen-space center of the puzzle board.
	cx := float32(rx) + float32(bw)/2
	cy := float32(boardTop) + float32(bh)/2

	splashColors := []color.RGBA{
		{255, 220, 50, 255},  // Gold
		{255, 100, 80, 255},  // Coral
		{100, 200, 255, 255}, // Sky blue
		{180, 100, 255, 255}, // Purple
		{80, 230, 120, 255},  // Green
		{255, 160, 40, 255},  // Orange
	}

	for i := 0; i < 80; i++ {
		angle := g.rng.Float64() * math.Pi * 2
		speed := float32(2.5 + g.rng.Float64()*5.5)
		life := 40 + g.rng.Intn(40)
		c := splashColors[g.rng.Intn(len(splashColors))]
		g.particles = append(g.particles, VictoryParticle{
			// Spawn slightly off-center for a more natural burst origin.
			x:       cx + float32(g.rng.Float64()-0.5)*float32(bw)*0.3,
			y:       cy + float32(g.rng.Float64()-0.5)*float32(bh)*0.3,
			vx:      float32(math.Cos(angle)) * speed,
			vy:      float32(math.Sin(angle)) * speed * 0.7, // flatten vertical spread slightly
			life:    life,
			maxLife: life,
			size:    float32(3 + g.rng.Intn(5)),
			color:   c,
		})
	}
}

// ── Draw ──────────────────────────────────────────────────────────────────────

// Draw is called by Ebitengine once per frame to render the current game state.
// It clears the screen and delegates to the appropriate draw function.
func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case StateMenu:
		g.drawMenu(screen)
	case StatePlaying:
		g.drawGame(screen)
	case StateWon:
		// Draw the completed game board first, then layer the win overlay on top.
		g.drawGame(screen)
		g.drawWin(screen)
	}
}

// drawMenu renders the main menu screen, including the decorative tile row,
// title, instructions, grid size selector buttons, and start prompt.
func (g *Game) drawMenu(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 18, 28, 255})

	// ── Decorative tile row ───────────────────────────────────────────────
	// Eight colored tiles are drawn across the top, centered horizontally.
	const tileW = 44
	const tileGap = 6
	const numTiles = 8
	const totalTilesW = numTiles*tileW + (numTiles-1)*tileGap
	tileStartX := float32(windowW/2) - float32(totalTilesW)/2

	for i, c := range tileColors[:numTiles] {
		x := tileStartX + float32(i)*(tileW+tileGap)
		vector.DrawFilledRect(screen, x, 30, tileW, tileW, c, false)
		// Bevel: lighter top/left edges, darker bottom/right edges.
		light := lighten(c, 60)
		dark := darken(c, 50)
		vector.DrawFilledRect(screen, x, 30, tileW, 4, light, false)
		vector.DrawFilledRect(screen, x, 30, 4, tileW, light, false)
		vector.DrawFilledRect(screen, x, 30+tileW-4, tileW, 4, dark, false)
		vector.DrawFilledRect(screen, x+tileW-4, 30, 4, tileW, dark, false)
	}

	// ── Title ─────────────────────────────────────────────────────────────
	title := "COLOR  SLIDING  PUZZLE"
	ebitenutil.DebugPrintAt(screen, title, windowW/2-len(title)*7/2, 100)

	// Thin horizontal divider beneath the title.
	vector.StrokeLine(screen, float32(windowW/2-180), 120, float32(windowW/2+180), 120, 1, color.RGBA{80, 80, 100, 255}, false)

	// ── Instructions block ────────────────────────────────────────────────
	// Text is left-aligned within a block that is itself centered on screen.
	const blockX = windowW/2 - 160
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
		"N           —  new game (same size)",
		"ESC         —  return to this menu",
		"",
		"GRID SIZE",
		"",
		"3  —  3x3  (easy)",
		"4  —  4x4  (medium)",
		"5  —  5x5  (hard)",
	}
	for i, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, blockX, 138+i*16)
	}

	// ── Grid size selector ────────────────────────────────────────────────
	// Three buttons are rendered side by side, centered horizontally.
	// The button matching menuSize is highlighted in blue.
	const btnW = 80
	const btnGap = 10
	const numBtns = 3
	const totalBtnsW = numBtns*btnW + (numBtns-1)*btnGap
	btnStartX := float32(windowW/2) - float32(totalBtnsW)/2

	sizes := []struct {
		n     int
		label string
	}{{3, "3x3"}, {4, "4x4"}, {5, "5x5"}}

	for i, s := range sizes {
		bx := btnStartX + float32(i)*(btnW+btnGap)
		by := float32(450)
		bg := color.RGBA{40, 40, 60, 255} // unselected
		if g.menuSize == s.n {
			bg = color.RGBA{80, 120, 220, 255} // selected (blue highlight)
		}
		vector.DrawFilledRect(screen, bx, by, btnW, 30, bg, false)
		label := fmt.Sprintf("[%d] %s", s.n, s.label)
		ebitenutil.DebugPrintAt(screen, label, int(bx)+btnW/2-len(label)*7/2, int(by)+10)
	}

	// ── Start prompt ──────────────────────────────────────────────────────
	startMsg := "Press ENTER to start"
	ebitenutil.DebugPrintAt(screen, startMsg, windowW/2-len(startMsg)*7/2, 500)
}

// drawGame renders the active puzzle screen: background, both boards, board
// labels, the stats panel, and the footer hint. It is also called as a base
// layer when rendering the win overlay in StateWon.
func (g *Game) drawGame(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 18, 28, 255})

	lx := leftBoardOriginX(g.size)
	rx := rightBoardOriginX(g.size)
	ts := tileSize(g.size)

	// Determine which tile the cursor is hovering over on the puzzle board.
	// Hover highlighting is suppressed in StateWon so the completed board looks clean.
	mx, my := ebiten.CursorPosition()
	hoverRow, hoverCol := -1, -1
	if g.state == StatePlaying {
		hoverRow, hoverCol = pixelToTile(mx, my, rx, g.size)
	}

	// Draw goal board (no hover, not interactive).
	g.drawBoard(screen, g.goal, lx, ts, -1, -1)
	// Draw puzzle board (interactive, with hover highlight).
	g.drawBoard(screen, g.current, rx, ts, hoverRow, hoverCol)

	// ── Board labels ──────────────────────────────────────────────────────
	bw := float32(boardWidth(g.size))
	ebitenutil.DebugPrintAt(screen, "G O A L", lx+int(bw/2)-24, boardTop-44)
	ebitenutil.DebugPrintAt(screen, "P U Z Z L E", rx+int(bw/2)-36, boardTop-44)

	// ── Stats panel ───────────────────────────────────────────────────────
	// Displayed below the puzzle board. elapsed is frozen on win; live while playing.
	elapsed := g.elapsed
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	statsY := boardTop + g.size*(ts+gap) + 14
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Time:  %02d:%02d", mins, secs), rx, statsY)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Moves: %d", g.moves), rx, statsY+16)

	// ── Footer ────────────────────────────────────────────────────────────
	ebitenutil.DebugPrintAt(screen, "N = new game   ESC = menu", lx, windowH-20)
}

// drawWin renders the win overlay on top of the completed game board.
// It shows the solve message, then draws the victory particle burst.
func (g *Game) drawWin(screen *ebiten.Image) {
	elapsed := g.elapsed
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60

	// Centered solve summary at the top of the screen.
	msg := fmt.Sprintf("SOLVED!   %d moves   %02d:%02d", g.moves, mins, secs)
	ebitenutil.DebugPrintAt(screen, msg, windowW/2-len(msg)*3, boardTop-58)
	ebitenutil.DebugPrintAt(screen, "N = new game   ESC = menu", windowW/2-80, boardTop-42)

	// Draw each live particle, fading it out as its life drains.
	for _, p := range g.particles {
		if p.life <= 0 {
			continue
		}
		// Alpha is proportional to remaining lifetime fraction.
		alpha := float32(p.life) / float32(p.maxLife)
		c := color.RGBA{p.color.R, p.color.G, p.color.B, uint8(alpha * 255)}
		// Radius also shrinks with remaining life for a natural fade.
		vector.DrawFilledCircle(screen, p.x, p.y, p.size*alpha, c, false)
	}
}

// drawBoard renders a single size×size board at the given boardX origin.
// hoverRow and hoverCol identify the tile under the cursor; pass -1 to disable
// hover highlighting. Non-blank tiles receive a bevel effect (lighter top-left
// edges, darker bottom-right edges) to give a subtle raised appearance.
func (g *Game) drawBoard(screen *ebiten.Image, board [][]*Tile, boardX, ts, hoverRow, hoverCol int) {
	for row := 0; row < g.size; row++ {
		for col := 0; col < g.size; col++ {
			tile := board[row][col]
			x := float32(boardX + col*(ts+gap))
			y := float32(boardTop + row*(ts+gap))
			tsf := float32(ts)

			if tile.IsBlank {
				// The blank cell is drawn as a dark recess with no bevel.
				vector.DrawFilledRect(screen, x, y, tsf, tsf, color.RGBA{35, 35, 45, 255}, false)
				continue
			}

			// Draw a white highlight border around tiles that are adjacent to the
			// blank and are therefore valid click targets.
			if row == hoverRow && col == hoverCol && isAdjacent(board, row, col) {
				vector.DrawFilledRect(screen, x-3, y-3, tsf+6, tsf+6, color.RGBA{255, 255, 255, 200}, false)
			}

			// Main tile body.
			tileColor := tile.Color
			vector.DrawFilledRect(screen, x, y, tsf, tsf, tileColor, false)

			// Bevel: 4-pixel strips along each edge.
			dark := darken(tileColor, 50)
			light := lighten(tileColor, 60)
			vector.DrawFilledRect(screen, x, y+tsf-4, tsf, 4, dark, false) // bottom edge
			vector.DrawFilledRect(screen, x+tsf-4, y, 4, tsf, dark, false) // right edge
			vector.DrawFilledRect(screen, x, y, tsf, 4, light, false)      // top edge
			vector.DrawFilledRect(screen, x, y, 4, tsf, light, false)      // left edge
		}
	}
}

// Layout implements ebiten.Game. It returns the logical screen dimensions,
// which Ebitengine uses to scale the game to the actual window size.
func (g *Game) Layout(_, _ int) (int, int) {
	return windowW, windowH
}

// ── Color helpers ─────────────────────────────────────────────────────────────

// darken returns a copy of c with each RGB channel reduced by d, clamped to 0.
// Used to compute the shadow edge of the tile bevel effect.
func darken(c color.RGBA, d uint8) color.RGBA {
	sub := func(a, b uint8) uint8 {
		if a < b {
			return 0
		}
		return a - b
	}
	return color.RGBA{sub(c.R, d), sub(c.G, d), sub(c.B, d), c.A}
}

// lighten returns a copy of c with each RGB channel increased by d, clamped to 255.
// Used to compute the highlight edge of the tile bevel effect.
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

// main is the program entry point. It configures the Ebitengine window and
// starts the game loop beginning at the main menu (StateMenu).
func main() {
	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowTitle("Color Sliding Puzzle")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	if err := ebiten.RunGame(initialGame()); err != nil {
		log.Fatal(err)
	}
}
