package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChizhovVadim/CounterGo/common"
)

const (
	GameResultWhiteWin = "1-0"
	GameResultBlackWin = "0-1"
	GameResultDraw     = "1/2-1/2"
)

type Tag struct {
	Key   string
	Value string
}

type Comment struct {
	Depth int
	Score common.UciScore
}

func loadPgnsManyFiles(ctx context.Context, files []string, pgns chan<- string) error {
	for _, filepath := range files {
		var err = loadPgns(ctx, filepath, pgns)
		if err != nil {
			return err
		}
	}
	return nil
}

// empty line beetwen tag-moves, beetwen games.
func loadPgns(ctx context.Context, filepath string, pgns chan<- string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	var sb = &strings.Builder{}
	var emptyLines int

	var scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		var line = scanner.Text()
		if line == "" {
			emptyLines++
			if emptyLines >= 2 && sb.Len() > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case pgns <- sb.String():
					sb = &strings.Builder{}
					emptyLines = 0
				}
			}
		} else {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return scanner.Err()
}

func parseTags(pgn string) []Tag {
	var tags = make([]Tag, 0, 16)
	tagMatches := tagPairRegex.FindAllStringSubmatch(pgn, -1)
	for i := range tagMatches {
		tags = append(tags, Tag{Key: tagMatches[i][1], Value: tagMatches[i][2]})
	}
	return tags
}

func tagValue(tags []Tag, key string) (string, bool) {
	for _, tag := range tags {
		if tag.Key == key {
			return tag.Value, true
		}
	}
	return "", false
}

func sanMovesFromPgn(pgn string) []string {
	pgn = commentsRegex.ReplaceAllString(pgn, "")
	pgn = tagsRegex.ReplaceAllString(pgn, "")
	pgn = moveNumberRegex.ReplaceAllString(pgn, "")
	return strings.Fields(pgn)
}

func parseComment(comment string) (Comment, error) {
	comment = strings.TrimLeft(comment, "{")
	comment = strings.TrimRight(comment, "}")
	var fields = strings.Fields(comment)
	if len(fields) >= 2 {
		var s string
		if strings.HasPrefix(fields[0], "(") {
			s = fields[1]
		} else {
			s = fields[0]
		}
		if s != "" {
			var index = strings.Index(s, "/")
			if index >= 0 {
				var sScore = s[:index]
				var sDepth = s[index+1:]

				var uciScore common.UciScore
				if strings.Contains(sScore, "M") {
					sScore = strings.Replace(sScore, "M", "", 1)
					score, err := strconv.Atoi(sScore)
					if err != nil {
						return Comment{}, err
					}
					uciScore = common.UciScore{Mate: score}
				} else {
					score, err := strconv.ParseFloat(sScore, 64)
					if err != nil {
						return Comment{}, err
					}
					uciScore = common.UciScore{Centipawns: int(100 * score)}
				}

				depth, err := strconv.Atoi(sDepth)
				if err != nil {
					return Comment{}, err
				}
				return Comment{
					Score: uciScore,
					Depth: depth,
				}, nil
			}
		}
	}
	return Comment{}, fmt.Errorf("parseComment %v", comment)
}

func commentsFromPgn(pgn string) []Comment {
	var result = make([]Comment, 0, 128)
	var comments = commentsRegex.FindAllString(pgn, -1)
	for i := range comments {
		var comment, err = parseComment(comments[i])
		if err != nil {
			break
		}
		result = append(result, comment)
	}
	return result
}

var startPosition, _ = common.NewPositionFromFEN(common.InitialPositionFen)

var (
	commentsRegex   = regexp.MustCompile(`{[^}]+}`)
	tagsRegex       = regexp.MustCompile(`\[[^\]]+\]`)
	moveNumberRegex = regexp.MustCompile(`[[:digit:]]+\.[[:space:]]`)
	tagPairRegex    = regexp.MustCompile(`\[(.*)\s\"(.*)\"\]`)
)
