package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Color sliding puzzle — GUI version using Ebitengine.
// Left board = goal. Right board = puzzle to solve.
// Click a tile adjacent to the blank to slide it. Press N for a new game.

const (
	windowW  = 850
	windowH  = 480
	tileSize = 110
	gap      = 6
	boardTop = 70 // y pixel where boards start
)

// leftBoardX is the x origin of the goal board.
// rightBoardX is the x origin of the playable board.
const (
	leftBoardX  = 30
	rightBoardX = leftBoardX + 3*(tileSize+gap) + 80
)

var tileColors = []color.RGBA{
	{220, 60, 60, 255},   // Red
	{60, 190, 60, 255},   // Green
	{60, 90, 220, 255},   // Blue
	{190, 60, 190, 255},  // Magenta
	{50, 205, 205, 255},  // Cyan
	{220, 190, 40, 255},  // Yellow
	{200, 200, 200, 255}, // Light gray
	{230, 110, 30, 255},  // Orange
}

// Tile is one cell of the puzzle.
type Tile struct {
	Color      color.RGBA
	CorrectPos int // position index in the solved board
	IsBlank    bool
}

func createTiles() []*Tile {
	tiles := make([]*Tile, 0, 9)
	for i := 0; i < 8; i++ {
		tiles = append(tiles, &Tile{Color: tileColors[i], CorrectPos: i})
	}
	tiles = append(tiles, &Tile{IsBlank: true, CorrectPos: 8})
	return tiles
}

func createBoard(tiles []*Tile) [][]*Tile {
	b := make([][]*Tile, 3)
	for i := range b {
		b[i] = make([]*Tile, 3)
		for j := range b[i] {
			b[i][j] = tiles[i*3+j]
		}
	}
	return b
}

func cloneBoard(src [][]*Tile) [][]*Tile {
	dst := make([][]*Tile, 3)
	for i := range src {
		dst[i] = make([]*Tile, 3)
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

func shuffleSolvable(tiles []*Tile) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		for i := len(tiles) - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			tiles[i], tiles[j] = tiles[j], tiles[i]
		}
		if countInversions(tiles)%2 == 0 {
			break
		}
	}
}

func findBlank(board [][]*Tile) (int, int) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
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
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if current[i][j] != goal[i][j] {
				return false
			}
		}
	}
	return true
}

func shuffleBoard(board [][]*Tile, moves int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + 1))
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for i := 0; i < moves; i++ {
		br, bc := findBlank(board)
		d := dirs[rng.Intn(4)]
		nr, nc := br+d[0], bc+d[1]
		if nr >= 0 && nr < 3 && nc >= 0 && nc < 3 {
			board[br][bc], board[nr][nc] = board[nr][nc], board[br][bc]
		}
	}
}

// pixelToTile maps a screen coordinate to a (row, col) for a board at originX.
// Returns (-1,-1) if the point is outside the board.
func pixelToTile(px, py, originX int) (int, int) {
	lx := px - originX
	ly := py - boardTop
	if lx < 0 || ly < 0 {
		return -1, -1
	}
	col := lx / (tileSize + gap)
	row := ly / (tileSize + gap)
	if col >= 3 || row >= 3 {
		return -1, -1
	}
	// Exclude gap pixels
	if lx-col*(tileSize+gap) >= tileSize || ly-row*(tileSize+gap) >= tileSize {
		return -1, -1
	}
	return row, col
}

// tileOrigin returns the top-left screen pixel for a tile at (row, col) of a board.
func tileOrigin(row, col, boardX int) (float32, float32) {
	x := float32(boardX + col*(tileSize+gap))
	y := float32(boardTop + row*(tileSize+gap))
	return x, y
}

// Game states
type State int

const (
	StatePlaying State = iota
	StateWon
)

// Game implements ebiten.Game.
type Game struct {
	goal    [][]*Tile
	current [][]*Tile
	moves   int
	state   State
}

func newGame() *Game {
	tiles := createTiles()
	shuffleSolvable(tiles)
	goal := createBoard(tiles)
	current := cloneBoard(goal)
	shuffleBoard(current, 120)
	// Make sure we didn't accidentally start solved
	for checkWin(current, goal) {
		shuffleBoard(current, 30)
	}
	return &Game{goal: goal, current: current}
}

func (g *Game) Update() error {
	// New game
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		*g = *newGame()
		return nil
	}

	if g.state != StatePlaying {
		return nil
	}

	// Click handling
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		row, col := pixelToTile(mx, my, rightBoardX)
		if row >= 0 && col >= 0 {
			if trySlide(g.current, row, col) {
				g.moves++
				if checkWin(g.current, g.goal) {
					g.state = StateWon
				}
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Background
	screen.Fill(color.RGBA{18, 18, 28, 255})

	mx, my := ebiten.CursorPosition()
	hoverRow, hoverCol := pixelToTile(mx, my, rightBoardX)

	// Draw both boards
	g.drawBoard(screen, g.goal, leftBoardX, -1, -1)
	g.drawBoard(screen, g.current, rightBoardX, hoverRow, hoverCol)

	// Board labels
	ebitenutil.DebugPrintAt(screen, "      G O A L", leftBoardX+28, boardTop-40)
	ebitenutil.DebugPrintAt(screen, "    P U Z Z L E", rightBoardX+18, boardTop-40)

	// Moves
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Moves: %d", g.moves), rightBoardX, boardTop+3*(tileSize+gap)+16)

	// Instructions
	ebitenutil.DebugPrintAt(screen, "Click a tile next to the blank to slide it   |   N = New Game", 20, windowH-20)

	// Win overlay
	if g.state == StateWon {
		msg := fmt.Sprintf("*** SOLVED in %d moves! ***   Press N for a new game.", g.moves)
		ebitenutil.DebugPrintAt(screen, msg, leftBoardX, boardTop-60)
	}
}

// drawBoard renders a 3×3 board. hoverRow/hoverCol highlight a hoverable tile (-1 to disable).
func (g *Game) drawBoard(screen *ebiten.Image, board [][]*Tile, boardX, hoverRow, hoverCol int) {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			tile := board[row][col]
			x, y := tileOrigin(row, col, boardX)

			if tile.IsBlank {
				// Dark recess for blank
				vector.DrawFilledRect(screen, x, y, tileSize, tileSize, color.RGBA{35, 35, 45, 255}, false)
				continue
			}

			// Hover highlight: white border if adjacent to blank and on playable board
			if row == hoverRow && col == hoverCol &&
				!tile.IsBlank && isAdjacent(board, row, col) {
				vector.DrawFilledRect(screen, x-3, y-3, tileSize+6, tileSize+6, color.RGBA{255, 255, 255, 200}, false)
			}

			// Tile body
			vector.DrawFilledRect(screen, x, y, tileSize, tileSize, tile.Color, false)

			// Subtle inner shadow / bevel: slightly darker bottom-right edge
			dark := darken(tile.Color, 50)
			vector.DrawFilledRect(screen, x, y+tileSize-4, tileSize, 4, dark, false)
			vector.DrawFilledRect(screen, x+tileSize-4, y, 4, tileSize, dark, false)

			// Lighter top-left bevel
			light := lighten(tile.Color, 60)
			vector.DrawFilledRect(screen, x, y, tileSize, 4, light, false)
			vector.DrawFilledRect(screen, x, y, 4, tileSize, light, false)
		}
	}
}

func (g *Game) Layout(_, _ int) (int, int) {
	return windowW, windowH
}

// darken reduces RGB channels by d (clamped to 0).
func darken(c color.RGBA, d uint8) color.RGBA {
	sub := func(a, b uint8) uint8 {
		if a < b {
			return 0
		}
		return a - b
	}
	return color.RGBA{sub(c.R, d), sub(c.G, d), sub(c.B, d), c.A}
}

// lighten increases RGB channels by d (clamped to 255).
func lighten(c color.RGBA, d uint8) color.RGBA {
	add := func(a, b uint8) uint8 {
		if int(a)+int(b) > 255 {
			return 255
		}
		return a + b
	}
	return color.RGBA{add(c.R, d), add(c.G, d), add(c.B, d), c.A}
}

func main() {
	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowTitle("Color Sliding Puzzle")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
