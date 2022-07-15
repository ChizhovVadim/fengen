package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ChizhovVadim/CounterGo/common"
)

type Game struct {
	Pgn        string //for debug
	GameResult string
	Items      []Item
}

type Item struct {
	Position   common.Position
	SanMove    string //for debug
	Comment    Comment
	SkipReason string //for debug
}

func (item Item) String() string {
	return fmt.Sprintln(item.SanMove, item.Comment, item.SkipReason)
}

func analyzeGames(
	ctx context.Context,
	quietService IQuietService,
	pgns <-chan string,
	games chan<- Game,
) error {
	for pgn := range pgns {
		var game, err = AnalyzeGame(quietService, pgn)
		if err != nil {
			log.Println("AnalyzeGame error", err, pgn)
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case games <- game:
		}
	}
	return nil
}

func saveFens(
	ctx context.Context,
	games <-chan Game,
	filepath string,
) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	var gameCount int
	var positionCount int

	for game := range games {

		var gameResult float32
		switch game.GameResult {
		case GameResultWhiteWin:
			gameResult = 1
		case GameResultBlackWin:
			gameResult = 0
		case GameResultDraw:
			gameResult = 0.5
		}

		for i := range game.Items {
			var item = &game.Items[i]
			if item.SkipReason != "" {
				continue
			}

			var fen = item.Position.String()
			var score = item.Comment.Score.Centipawns
			// score from white point of view
			if !item.Position.WhiteMove {
				score = -score
			}
			_, err = fmt.Fprintf(file, "%v;%v;%v\n",
				fen,
				score,
				gameResult)
			if err != nil {
				return err
			}
			positionCount++
		}

		gameCount++
		if gameCount%1000 == 0 {
			log.Printf("Saved %v games, %v positions\n", gameCount, positionCount)
		}
	}

	log.Printf("Saved %v games, %v positions\n", gameCount, positionCount)
	return nil
}

const (
	SkipReasonRepeat        = "Repeat"
	SkipReasonNoAnalyze     = "NoAnalyze"
	SkipReasonCheckmateSoon = "CheckmateSoon"
	SkipReasonHiFiftyMove   = "HiFiftyMove"
	SkipReasonInCheck       = "InCheck"
	SkipReasonNoisy         = "Noisy"
)

func AnalyzeGame(quietService IQuietService, pgn string) (Game, error) {
	var tags = parseTags(pgn)
	if len(tags) == 0 {
		return Game{}, fmt.Errorf("empty tags")
	}

	var gameResult, gameResultOk = tagValue(tags, "Result")
	if !(gameResultOk &&
		(gameResult == GameResultWhiteWin ||
			gameResult == GameResultBlackWin ||
			gameResult == GameResultDraw)) {
		return Game{}, fmt.Errorf("bad game result %v", pgn)
	}
	var isDraw = gameResult == GameResultDraw

	var curPosition = startPosition
	var sanMoves = sanMovesFromPgn(pgn)
	var comments = commentsFromPgn(pgn)
	var capacity = min(256, len(sanMoves))
	var items = make([]Item, 0, capacity)
	var repeatPositions = make(map[uint64]struct{})

	for i, san := range sanMoves {
		var move = common.ParseMoveSAN(&curPosition, san)
		if move == common.MoveEmpty {
			break
		}
		var child common.Position
		if !curPosition.MakeMove(move, &child) {
			break
		}

		if i >= len(comments) {
			break
		}
		var comment = comments[i]

		var skipReason string
		if comment.Depth < 10 {
			skipReason = SkipReasonNoAnalyze
		} else if comment.Score.Mate != 0 {
			skipReason = SkipReasonCheckmateSoon
		} else if curPosition.IsCheck() {
			skipReason = SkipReasonInCheck
		} else if curPosition.Rule50 >= 40 && isDraw {
			skipReason = SkipReasonHiFiftyMove
		} else if _, found := repeatPositions[curPosition.Key]; found {
			skipReason = SkipReasonRepeat
		} else if !quietService.IsQuiet(&curPosition) {
			skipReason = SkipReasonNoisy
		}

		items = append(items, Item{
			SanMove:    san,
			Position:   curPosition,
			Comment:    comment,
			SkipReason: skipReason,
		})

		repeatPositions[curPosition.Key] = struct{}{}
		curPosition = child
	}

	return Game{
		Pgn:        pgn,
		GameResult: gameResult,
		Items:      items,
	}, nil
}
