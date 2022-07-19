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

type Game struct {
	Tags  []Tag
	Items []Item
}

type Tag struct {
	Key   string
	Value string
}

type Item struct {
	SanMove    string //for debug
	TxtComment string //for debug
	Position   common.Position
	Comment    Comment
}

func (item Item) String() string {
	return fmt.Sprintln(item.SanMove, item.TxtComment, item.Comment)
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

func ParseGame(pgn string) (Game, error) {
	var tags = parseTags(pgn)

	if len(tags) == 0 {
		return Game{}, fmt.Errorf("empty tags")
	}

	var sanMoves = sanMovesFromPgn(pgn)
	var comments = commentsFromPgn(pgn)

	if len(sanMoves)-1 == len(comments) {
		//remove game result
		sanMoves = sanMoves[:len(sanMoves)-1]
	}
	if len(sanMoves) != len(comments) {
		return Game{}, fmt.Errorf("inconsistent moves and comments %v %v", len(sanMoves), len(comments))
	}

	var curPosition = startPosition
	var items = make([]Item, 0, len(sanMoves))

	for i, san := range sanMoves {
		var move = common.ParseMoveSAN(&curPosition, san)
		if move == common.MoveEmpty {
			break
		}
		var child common.Position
		if !curPosition.MakeMove(move, &child) {
			break
		}

		var txtComment = comments[i]
		var comment, _ = parseComment(txtComment)

		items = append(items, Item{
			SanMove:    san,
			TxtComment: txtComment,
			Position:   curPosition,
			Comment:    comment,
		})

		curPosition = child
	}

	if len(items) == 0 {
		return Game{}, fmt.Errorf("no moves")
	}

	return Game{
		Tags:  tags,
		Items: items,
	}, nil
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

	if comment == "book" {
		return Comment{}, nil
	}

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

func commentsFromPgn(pgn string) []string {
	var comments = commentsRegex.FindAllString(pgn, -1)
	return comments
}

var startPosition, _ = common.NewPositionFromFEN(common.InitialPositionFen)

var (
	commentsRegex   = regexp.MustCompile(`{[^}]+}`)
	tagsRegex       = regexp.MustCompile(`\[[^\]]+\]`)
	moveNumberRegex = regexp.MustCompile(`[[:digit:]]+\.[[:space:]]`)
	tagPairRegex    = regexp.MustCompile(`\[(.*)\s\"(.*)\"\]`)
)
