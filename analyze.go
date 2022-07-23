package main

import (
	"context"
	"fmt"
	"log"
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
		if len(game) != 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case games <- game:
			}
		}
	}
	return nil
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
		if i > 0 {
			repeatPositions[game.Items[i-1].Position.Key] = struct{}{}
		}

		var item = &game.Items[i]

		if item.Comment.Depth < 10 ||
			item.Comment.Score.Mate != 0 ||
			item.Position.IsCheck() {
			continue
		}
		if _, found := repeatPositions[item.Position.Key]; found {
			continue
		}
		if !quietService.IsQuiet(&item.Position) {
			continue
		}

		result = append(result, PositionInfo{
			position:   item.Position,
			score:      item.Comment.Score.Centipawns,
			gameResult: gameResult,
		})
	}

	return result, nil
}
