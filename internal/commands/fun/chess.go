package fun

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/notnil/chess"
	"skyvern/internal/config"
	"skyvern/internal/manager"
)

type ChessGame struct {
	Game      *chess.Game
	WhiteID   string
	BlackID   string
	Level     int
	LastMove  string
}

type ChessChallenge struct {
	ChallengerID string
	OpponentID   string
	ChannelID    string
	ExpiresAt    time.Time
}

func hasCheck(g *chess.Game) bool {
	moves := g.Moves()
	if len(moves) == 0 {
		return false
	}
	return moves[len(moves)-1].HasTag(chess.Check)
}

var (
	activeGames         = make(map[string]*ChessGame)
	activeGamesMu       sync.Mutex
	pendingChallenges   = make(map[string]*ChessChallenge)
	pendingChallengesMu sync.Mutex
)

func init() {
	manager.RegisterHelp("chess", []manager.HelpPage{
		{
			Command:     "Chess Play AI",
			Syntax:      ".chess play [level] [color]",
			Description: "Start a chess game against Stockfish (level 0-20, color white/black).",
		},
		{
			Command:     "Chess Play PvP",
			Syntax:      ".chess play <@opponent>",
			Description: "Challenge another player to a chess game.",
		},
		{
			Command:     "Chess Accept",
			Syntax:      ".chess accept",
			Description: "Accept a pending chess challenge.",
		},
		{
			Command:     "Chess Decline",
			Syntax:      ".chess decline",
			Description: "Decline a pending chess challenge.",
		},
		{
			Command:     "Chess Move",
			Syntax:      ".chess move <SAN/Coordinate>",
			Description: "Make a move on the active board (e.g. e4, Nf3).",
		},
		{
			Command:     "Chess Board",
			Syntax:      ".chess board",
			Description: "Display the active chess board.",
		},
		{
			Command:     "Chess Resign",
			Syntax:      ".chess resign",
			Description: "Resign/stop the active chess game.",
		},
	})
}

func getStockfishURL() string {
	osVal := runtime.GOOS
	archVal := runtime.GOARCH
	switch osVal {
	case "windows":
		return "https://github.com/official-stockfish/Stockfish/releases/download/sf_18/stockfish-windows-x86-64.zip"
	case "darwin":
		if archVal == "arm64" {
			return "https://github.com/official-stockfish/Stockfish/releases/download/sf_18/stockfish-macos-m1-apple-silicon.tar"
		}
		return "https://github.com/official-stockfish/Stockfish/releases/download/sf_18/stockfish-macos-x86-64.tar"
	case "linux":
		return "https://github.com/official-stockfish/Stockfish/releases/download/sf_18/stockfish-ubuntu-x86-64.tar"
	case "android":
		return "https://github.com/official-stockfish/Stockfish/releases/download/sf_18/stockfish-android-armv8.tar"
	}
	return ""
}

func downloadAndExtractStockfish(destPath string, url string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", res.Status)
	}
	tempFile := destPath + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	f.Close()
	if err != nil {
		os.Remove(tempFile)
		return err
	}
	defer os.Remove(tempFile)

	if strings.HasSuffix(url, ".zip") {
		zr, err := zip.OpenReader(tempFile)
		if err != nil {
			return err
		}
		defer zr.Close()
		for _, file := range zr.File {
			if file.FileInfo().IsDir() {
				continue
			}
			lowerName := strings.ToLower(file.Name)
			if strings.Contains(lowerName, "stockfish") && strings.HasSuffix(lowerName, ".exe") {
				rc, err := file.Open()
				if err != nil {
					return err
				}
				defer rc.Close()
				outF, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
				if err != nil {
					return err
				}
				defer outF.Close()
				_, err = io.Copy(outF, rc)
				return err
			}
		}
		return fmt.Errorf("no stockfish binary found in zip")
	} else if strings.HasSuffix(url, ".tar") {
		tarFile, err := os.Open(tempFile)
		if err != nil {
			return err
		}
		defer tarFile.Close()
		tr := tar.NewReader(tarFile)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if hdr.Typeflag == tar.TypeDir {
				continue
			}
			lowerName := strings.ToLower(hdr.Name)
			if strings.Contains(lowerName, "stockfish") && !strings.Contains(lowerName, ".txt") && !strings.Contains(lowerName, ".md") {
				outF, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
				if err != nil {
					return err
				}
				defer outF.Close()
				_, err = io.Copy(outF, tr)
				return err
			}
		}
		return fmt.Errorf("no stockfish binary found in tar")
	}
	return fmt.Errorf("unsupported archive format")
}

func getStockfishMove(stockfishPath string, fen string, skillLevel int) (string, error) {
	cmd := exec.Command(stockfishPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	defer cmd.Process.Kill()
	fmt.Fprintf(stdin, "uci\n")
	fmt.Fprintf(stdin, "setoption name Skill Level value %d\n", skillLevel)
	fmt.Fprintf(stdin, "position fen %s\n", fen)
	fmt.Fprintf(stdin, "go movetime 500\n")
	stdin.Close()
	scanner := bufio.NewScanner(stdout)
	bestMove := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "bestmove ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				bestMove = parts[1]
			}
			break
		}
	}
	_ = cmd.Wait()
	if bestMove == "" {
		return "", fmt.Errorf("no bestmove returned")
	}
	return bestMove, nil
}

func renderChessBoard(fen string, lastMove string, inCheck bool) ([]byte, error) {
	inCheckStr := "false"
	if inCheck {
		inCheckStr = "true"
	}
	lastMoveStr := "null"
	if lastMove != "" {
		lastMoveStr = fmt.Sprintf("'%s'", lastMove)
	}
	jsCode := fmt.Sprintf(`
const { renderBoard } = require('./src/utils/chessRenderer');
try {
	const buf = renderBoard('%s', %s, %s);
	console.log(buf.toString('base64'));
} catch (e) {
	console.error(e);
	process.exit(1);
}
`, fen, lastMoveStr, inCheckStr)

	nodeBin := resolveNodePath()
	cmd := exec.Command(nodeBin, "-e", jsCode)
	dir := "../percwtf4"
	if _, err := os.Stat(dir); err != nil {
		dir = "C:\\Users\\vir\\Documents\\percs1\\n\\prc\\percwtf4"
		if _, err := os.Stat(dir); err != nil {
			dir = "."
		}
	}
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("node failed: %v, stderr: %s", err, stderr.String())
	}
	trimmed := strings.TrimSpace(stdout.String())
	return base64.StdEncoding.DecodeString(trimmed)
}

func resolveNodePath() string {
	if p, err := exec.LookPath("node"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		commonPaths := []string{
			`C:\Program Files\nodejs\node.exe`,
			`C:\Program Files (x86)\nodejs\node.exe`,
		}
		for _, cp := range commonPaths {
			if _, err := os.Stat(cp); err == nil {
				return cp
			}
		}
	}
	return "node"
}

func formatBoard(board *chess.Board, playerColor chess.Color) string {
	boardStr := board.Draw()
	rows := strings.Split(strings.TrimSpace(boardStr), "\n")
	var sb strings.Builder
	sb.WriteString("```\n")
	if playerColor == chess.Black {
		sb.WriteString("  h g f e d c b a\n")
		for i := 7; i >= 0; i-- {
			rowRunes := strings.Fields(rows[i])
			rank := 8 - i
			sb.WriteString(fmt.Sprintf("%d ", rank))
			for j := 7; j >= 0; j-- {
				sb.WriteString(rowRunes[j] + " ")
			}
			sb.WriteString(fmt.Sprintf("%d\n", rank))
		}
		sb.WriteString("  h g f e d c b a\n")
	} else {
		sb.WriteString("  a b c d e f g h\n")
		for i := 0; i < 8; i++ {
			rank := 8 - i
			sb.WriteString(fmt.Sprintf("%d ", rank))
			rowRunes := strings.Fields(rows[i])
			for _, r := range rowRunes {
				sb.WriteString(r + " ")
			}
			sb.WriteString(fmt.Sprintf("%d\n", rank))
		}
		sb.WriteString("  a b c d e f g h\n")
	}
	sb.WriteString("```")
	return sb.String()
}

func sendBoardEmbed(ctx *manager.CommandContext, game *ChessGame) error {
	inCheck := hasCheck(game.Game)
	pngBytes, err := renderChessBoard(game.Game.Position().String(), game.LastMove, inCheck)
	if err != nil {
		boardStr := formatBoard(game.Game.Position().Board(), chess.White)
		return ctx.Reply(fmt.Sprintf("[!] Renderer failed: %v. Text board:\n%s", err, boardStr))
	}
	turn := game.Game.Position().Turn()
	var turnText string
	var activePlayer string
	if turn == chess.White {
		activePlayer = game.WhiteID
		turnText = "White to move"
	} else {
		activePlayer = game.BlackID
		turnText = "Black to move"
	}
	if activePlayer == "stockfish" {
		turnText += " (Stockfish AI)"
	} else {
		turnText += fmt.Sprintf(" (<@%s>)", activePlayer)
	}
	whiteLabel := "Stockfish (AI)"
	if game.WhiteID != "stockfish" {
		whiteLabel = fmt.Sprintf("<@%s>", game.WhiteID)
	}
	blackLabel := "Stockfish (AI)"
	if game.BlackID != "stockfish" {
		blackLabel = fmt.Sprintf("<@%s>", game.BlackID)
	}
	colorHex := 0xf0d9b5
	if turn == chess.Black {
		colorHex = 0xb58863
	}
	embed := &discordgo.MessageEmbed{
		Title: "Chess",
		Color: colorHex,
		Description: fmt.Sprintf(
			"**White:** %s\n**Black:** %s\n\n**Status:** %s",
			whiteLabel, blackLabel, turnText,
		),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://board.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Type \".chess move <move>\" or \".chess resign\"",
		},
	}
	ms := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Files: []*discordgo.File{
			{
				Name:   "board.png",
				Reader: bytes.NewReader(pngBytes),
			},
		},
	}
	_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
	return err
}

var Chess = &manager.Command{
	Trigger:     "chess",
	Aliases:     []string{"playchess"},
	Name:        "chess",
	Description: "Play chess against Stockfish 18 or another user",
	Category:    "fun",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("chess")
		}
		sub := strings.ToLower(ctx.Args[0])
		chanID := ctx.ChanID()
		authorID := ctx.AuthorID()
		destDir := config.ResolvePath("bin")
		_ = os.MkdirAll(destDir, 0755)
		binaryName := "stockfish"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		stockfishPath := filepath.Join(destDir, binaryName)

		switch sub {
		case "play", "start":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.chess play <@opponent>` or `.chess play [level] [color]`")
			}
			activeGamesMu.Lock()
			existing := activeGames[chanID]
			activeGamesMu.Unlock()
			if existing != nil {
				return ctx.Reply("[!] A chess game is already active in this channel.")
			}

			targetArg := ctx.Args[1]
			opponentID := ""
			if strings.HasPrefix(targetArg, "<@") && strings.HasSuffix(targetArg, ">") {
				opponentID = strings.Trim(targetArg, "<@!>")
			}

			if opponentID != "" {
				if opponentID == authorID {
					return ctx.Reply("[!] You cannot challenge yourself.")
				}
				pendingChallengesMu.Lock()
				pendingChallenges[authorID+":"+chanID] = &ChessChallenge{
					ChallengerID: authorID,
					OpponentID:   opponentID,
					ChannelID:    chanID,
					ExpiresAt:    time.Now().Add(5 * time.Minute),
				}
				pendingChallengesMu.Unlock()
				return ctx.Reply(fmt.Sprintf("<@%s>, <@%s> has challenged you to a game of chess! Type `.chess accept` to start or `.chess decline` to refuse.", opponentID, authorID))
			}

			if _, err := os.Stat(stockfishPath); os.IsNotExist(err) {
				url := getStockfishURL()
				if url == "" {
					return ctx.Reply("[!] Unsupported system for Stockfish.")
				}
				_ = ctx.Reply(fmt.Sprintf("[*] Stockfish binary not found. Downloading Stockfish 18 (%s)...", filepath.Base(url)))
				if err := downloadAndExtractStockfish(stockfishPath, url); err != nil {
					return ctx.Reply(fmt.Sprintf("[!] Download failed: %v", err))
				}
				_ = ctx.Reply("[+] Stockfish downloaded successfully.")
			}
			level := 5
			playerCol := chess.White
			for _, arg := range ctx.Args[1:] {
				argLower := strings.ToLower(arg)
				if argLower == "white" {
					playerCol = chess.White
				} else if argLower == "black" {
					playerCol = chess.Black
				} else if val, err := strconv.Atoi(arg); err == nil {
					if val >= 0 && val <= 20 {
						level = val
					}
				}
			}
			g := chess.NewGame()
			chessGame := &ChessGame{
				Game:    g,
				WhiteID: authorID,
				BlackID: "stockfish",
				Level:   level,
			}
			if playerCol == chess.Black {
				chessGame.WhiteID = "stockfish"
				chessGame.BlackID = authorID
			}
			activeGamesMu.Lock()
			activeGames[chanID] = chessGame
			activeGamesMu.Unlock()

			if playerCol == chess.Black {
				_ = ctx.Reply("[*] Stockfish is calculating the first move...")
				sfMove, err := getStockfishMove(stockfishPath, g.Position().String(), level)
				if err != nil {
					activeGamesMu.Lock()
					delete(activeGames, chanID)
					activeGamesMu.Unlock()
					return ctx.Reply(fmt.Sprintf("[!] Stockfish failed to make a move: %v", err))
				}
				if err := g.MoveStr(sfMove); err != nil {
					activeGamesMu.Lock()
					delete(activeGames, chanID)
					activeGamesMu.Unlock()
					return ctx.Reply(fmt.Sprintf("[!] Invalid Stockfish move: %v", err))
				}
				chessGame.LastMove = sfMove
			}
			return sendBoardEmbed(ctx, chessGame)

		case "accept":
			pendingChallengesMu.Lock()
			var challenge *ChessChallenge
			var challengerKey string
			for k, c := range pendingChallenges {
				if c.OpponentID == authorID && c.ChannelID == chanID && time.Now().Before(c.ExpiresAt) {
					challenge = c
					challengerKey = k
					break
				}
			}
			if challenge != nil {
				delete(pendingChallenges, challengerKey)
			}
			pendingChallengesMu.Unlock()

			if challenge == nil {
				return ctx.Reply("[!] No pending chess challenge found for you in this channel.")
			}

			activeGamesMu.Lock()
			existing := activeGames[chanID]
			activeGamesMu.Unlock()
			if existing != nil {
				return ctx.Reply("[!] A game is already active in this channel.")
			}

			g := chess.NewGame()
			chessGame := &ChessGame{
				Game:    g,
				WhiteID: challenge.ChallengerID,
				BlackID: challenge.OpponentID,
			}
			activeGamesMu.Lock()
			activeGames[chanID] = chessGame
			activeGamesMu.Unlock()

			_ = ctx.Reply(fmt.Sprintf("[+] Challenge accepted! <@%s> (White) vs <@%s> (Black) has started.", challenge.ChallengerID, challenge.OpponentID))
			return sendBoardEmbed(ctx, chessGame)

		case "decline":
			pendingChallengesMu.Lock()
			var challengerKey string
			var challengerID string
			for k, c := range pendingChallenges {
				if c.OpponentID == authorID && c.ChannelID == chanID {
					challengerKey = k
					challengerID = c.ChallengerID
					break
				}
			}
			if challengerKey != "" {
				delete(pendingChallenges, challengerKey)
			}
			pendingChallengesMu.Unlock()

			if challengerKey == "" {
				return ctx.Reply("[!] No pending chess challenge found for you in this channel.")
			}
			return ctx.Reply(fmt.Sprintf("[+] Challenge declined. <@%s> has been notified.", challengerID))

		case "move", "playmove":
			activeGamesMu.Lock()
			g := activeGames[chanID]
			activeGamesMu.Unlock()
			if g == nil {
				return ctx.Reply("[!] No active chess game in this channel. Start one with `.chess play`.")
			}
			turn := g.Game.Position().Turn()
			activePlayer := g.WhiteID
			if turn == chess.Black {
				activePlayer = g.BlackID
			}
			if activePlayer != authorID {
				return ctx.Reply("[!] It is not your turn.")
			}
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.chess move <move>`")
			}
			moveStr := ctx.Args[1]
			if err := g.Game.MoveStr(moveStr); err != nil {
				inCheck := hasCheck(g.Game)
				pngBytes, _ := renderChessBoard(g.Game.Position().String(), g.LastMove, inCheck)
				if pngBytes != nil {
					embed := &discordgo.MessageEmbed{
						Title:       "Chess",
						Color:       0xcc0000,
						Description: fmt.Sprintf("[!] Illegal or invalid move: `%s`.\nIt is still your turn.", moveStr),
						Image: &discordgo.MessageEmbedImage{
							URL: "attachment://board.png",
						},
					}
					ms := &discordgo.MessageSend{
						Embeds: []*discordgo.MessageEmbed{embed},
						Files: []*discordgo.File{
							{
								Name:   "board.png",
								Reader: bytes.NewReader(pngBytes),
							},
						},
					}
					_, _ = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
					return nil
				}
				boardStr := formatBoard(g.Game.Position().Board(), chess.White)
				return ctx.Reply(fmt.Sprintf("[!] Illegal or invalid move.\n\n%s", boardStr))
			}

			moves := g.Game.Moves()
			if len(moves) > 0 {
				lastMoveObj := moves[len(moves)-1]
				g.LastMove = lastMoveObj.S1().String() + lastMoveObj.S2().String()
				if lastMoveObj.Promo() != chess.NoPieceType {
					g.LastMove += lastMoveObj.Promo().String()
				}
			}

			if g.Game.Outcome() != chess.NoOutcome {
				inCheck := hasCheck(g.Game)
				pngBytes, _ := renderChessBoard(g.Game.Position().String(), g.LastMove, inCheck)
				outcome := g.Game.Outcome()
				method := g.Game.Method()
				activeGamesMu.Lock()
				delete(activeGames, chanID)
				activeGamesMu.Unlock()
				var res string
				if outcome == chess.WhiteWon {
					res = "White won by " + method.String()
				} else if outcome == chess.BlackWon {
					res = "Black won by " + method.String()
				} else {
					res = "Draw by " + method.String()
				}
				if pngBytes != nil {
					embed := &discordgo.MessageEmbed{
						Title:       "Chess - Game Over",
						Color:       0xf1c40f,
						Description: fmt.Sprintf("**Result:** %s", res),
						Image: &discordgo.MessageEmbedImage{
							URL: "attachment://board.png",
						},
					}
					ms := &discordgo.MessageSend{
						Embeds: []*discordgo.MessageEmbed{embed},
						Files: []*discordgo.File{
							{
								Name:   "board.png",
								Reader: bytes.NewReader(pngBytes),
							},
						},
					}
					_, _ = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
					return nil
				}
				boardStr := formatBoard(g.Game.Position().Board(), chess.White)
				return ctx.Reply(fmt.Sprintf("[*] Game Over!\n\n%s\n\nResult: **%s**", boardStr, res))
			}

			nextTurn := g.Game.Position().Turn()
			nextPlayer := g.WhiteID
			if nextTurn == chess.Black {
				nextPlayer = g.BlackID
			}

			if nextPlayer == "stockfish" {
				_ = ctx.Reply("[*] Stockfish is thinking...")
				sfMove, err := getStockfishMove(stockfishPath, g.Game.Position().String(), g.Level)
				if err != nil {
					return ctx.Reply(fmt.Sprintf("[!] Stockfish failed to make a move: %v", err))
				}
				if err := g.Game.MoveStr(sfMove); err != nil {
					return ctx.Reply(fmt.Sprintf("[!] Invalid Stockfish move: %v", err))
				}
				g.LastMove = sfMove

				if g.Game.Outcome() != chess.NoOutcome {
					inCheck := hasCheck(g.Game)
					pngBytes, _ := renderChessBoard(g.Game.Position().String(), g.LastMove, inCheck)
					outcome := g.Game.Outcome()
					method := g.Game.Method()
					activeGamesMu.Lock()
					delete(activeGames, chanID)
					activeGamesMu.Unlock()
					var res string
					if outcome == chess.WhiteWon {
						res = "White won by " + method.String()
					} else if outcome == chess.BlackWon {
						res = "Black won by " + method.String()
					} else {
						res = "Draw by " + method.String()
					}
					if pngBytes != nil {
						embed := &discordgo.MessageEmbed{
							Title:       "Chess - Game Over",
							Color:       0xf1c40f,
							Description: fmt.Sprintf("**Result:** %s", res),
							Image: &discordgo.MessageEmbedImage{
								URL: "attachment://board.png",
							},
						}
						ms := &discordgo.MessageSend{
							Embeds: []*discordgo.MessageEmbed{embed},
							Files: []*discordgo.File{
								{
									Name:   "board.png",
									Reader: bytes.NewReader(pngBytes),
								},
							},
						}
						_, _ = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
						return nil
					}
					boardStr := formatBoard(g.Game.Position().Board(), chess.White)
					return ctx.Reply(fmt.Sprintf("[*] Game Over!\n\n%s\n\nResult: **%s**", boardStr, res))
				}
			}

			return sendBoardEmbed(ctx, g)

		case "board", "show":
			activeGamesMu.Lock()
			g := activeGames[chanID]
			activeGamesMu.Unlock()
			if g == nil {
				return ctx.Reply("[!] No active chess game in this channel.")
			}
			return sendBoardEmbed(ctx, g)

		case "resign", "stop", "quit":
			activeGamesMu.Lock()
			g := activeGames[chanID]
			if g != nil {
				delete(activeGames, chanID)
			}
			activeGamesMu.Unlock()
			if g == nil {
				return ctx.Reply("[!] No active chess game in this channel.")
			}
			return ctx.Reply(fmt.Sprintf("[+] <@%s> resigned. Game over.", authorID))

		default:
			return ctx.SendHelp("chess")
		}
	},
}
