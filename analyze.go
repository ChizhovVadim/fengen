package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ChizhovVadim/CounterGo/common"
)

func analyzeGames(
	ctx context.Context,
	quietService IQuietService,
	pgns <-chan string,
	games chan<- []PositionInfo,
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
	games <-chan []PositionInfo,
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
		for i := range game {
			var item = &game[i]
			var fen = item.position.String()
			var score = item.score
			// score from white point of view
			if !item.position.WhiteMove {
				score = -score
			}
			_, err = fmt.Fprintf(file, "%v;%v;%v\n",
				fen,
				score,
				item.gameResult)
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

type PositionInfo struct {
	position   common.Position
	score      int
	gameResult float32
}

func AnalyzeGame(quietService IQuietService, pgn string) ([]PositionInfo, error) {
	var game, err = ParseGame(pgn)
	if err != nil {
		return nil, err
	}

	var sGameResult, gameResultOk = tagValue(game.Tags, "Result")
	if !gameResultOk {
		return nil, fmt.Errorf("bad game result")
	}

	var gameResult float32
	switch sGameResult {
	case GameResultWhiteWin:
		gameResult = 1
	case GameResultBlackWin:
		gameResult = 0
	case GameResultDraw:
		gameResult = 0.5
	default:
		return nil, fmt.Errorf("bad game result")
	}

	var repeatPositions = make(map[uint64]struct{})
	var result []PositionInfo

	for i := range game.Items {
		var item = &game.Items[i]

		if !skipPosition(quietService, item, repeatPositions) {
			result = append(result, PositionInfo{
				position:   item.Position,
				score:      item.Comment.Score.Centipawns,
				gameResult: gameResult,
			})
		}

		repeatPositions[item.Position.Key] = struct{}{}
	}

	return result, nil
}

func skipPosition(quietService IQuietService,
	item *Item,
	repeatPositions map[uint64]struct{}) bool {

	var curPosition = &item.Position
	var comment = &item.Comment

	if comment.Depth < 10 ||
		comment.Score.Mate != 0 ||
		curPosition.IsCheck() {
		return true
	}

	if _, found := repeatPositions[curPosition.Key]; found {
		return true
	}
	if !quietService.IsQuiet(curPosition) {
		return true
	}

	return false
}
