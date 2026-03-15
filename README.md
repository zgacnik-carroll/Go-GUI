# Color Sliding Puzzle (Go GUI Game)

---

## Description

The Color Sliding Puzzle is a GUI application game written in Go using the [Ebitengine](https://ebitengine.org/) 2D game library. The player is presented with two boards side by side: a **Goal** board showing the solved arrangement of colored tiles, and a **Puzzle** board that has been scrambled. The player must slide the puzzle tiles into the correct positions to match the goal.

Three grid sizes are supported — 3×3, 4×4, and 5×5 — selectable from the main menu. A move counter and elapsed timer track the player's performance, and a confetti particle burst plays upon solving the puzzle.

---

## Requirements

- Go 1.21 or later (run `go version` in a terminal to confirm)

---

## How to Play

When the program is running, the player is first greeted with the **main menu**, which displays instructions and a grid size selector. After choosing a size and pressing **ENTER**, the game begins.

Two boards are displayed side by side:

- **Left — Goal board:** the solved arrangement the player must recreate
- **Right — Puzzle board:** the scrambled board the player interacts with

The player's job is to click tiles adjacent to the blank space to slide them into position, rearranging the puzzle board until it matches the goal board on the left.

### Controls

| Input | Action |
|---|---|
| **Click a tile** | Slide it into the adjacent blank space |
| **N** | Start a new game (same grid size) |
| **ESC** | Return to the main menu |
| **3 / 4 / 5** | Select grid size on the main menu |
| **ENTER** | Start the game from the main menu |

---

## How to Run

1. Clone this GitHub repository into your desired directory.
2. Navigate to your desired directory, then navigate to the cloned repository and run the following command:

   ```bash
   go run main.go
   ```
Once you have run this command within your terminal, the game will be up and running!

---

## Tile Generation Feature

The most significant feature within this program is the tile generation and shuffling system. Below is an in-depth description of how it works:

Each tile in the puzzle is represented by a `Tile` struct, and the program creates a slice of pointers to `Tile` using `createTiles()`. Each tile pointer stores its color, whether it is blank, and its `CorrectPos` index — the position it occupies in the solved arrangement.

The 1D slice of tile pointers is then converted into a 2D board with `createBoard()`. Shuffling is handled in two ways:

- **`shuffleSolvable()`** — randomly permutes the tile pointers using a Fisher-Yates shuffle, retrying until the arrangement is mathematically solvable. Whether a puzzle is solvable depends on its inversion count and, for even-width grids, the row of the blank tile.
- **`shuffleBoard()`** — scrambles the board by performing a series of random legal moves from the solved state, guaranteeing solvability without any retry logic.

Because the board holds pointers to tiles rather than tile values, swapping tiles only rearranges which pointer occupies a given cell — no tile data is copied. Both the goal board and the puzzle board reference the same underlying `Tile` objects, so win detection is a simple pointer comparison rather than a value comparison.

---

## Future Improvements

With more time and effort, these are some improvements that could be made to this project:

- **Smooth tile animations** — interpolate tile positions during a slide rather than swapping instantly, for a more polished feel
- **Sound effects** — add audio for tile clicks, puzzle completion, and invalid move attempts using Ebitengine's built-in audio support
- **Persistent high scores** — save the player's best move count and time per grid size to a local file so records survive between sessions
- **Hint system** — highlight the next recommended tile to move for players who get stuck
- **Undo functionality** — allow the player to reverse their last move

---

## Closing Remarks

This Color Sliding Puzzle game was created to strengthen my understanding of Go, Ebitengine, pointer-based data structures, and GUI application development. It combines tile generation logic, solvability mathematics, randomization, and real-time rendering to create a clean and engaging puzzle game experience.

Have fun playing the Color Sliding Puzzle!