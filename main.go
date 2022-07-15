package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ChizhovVadim/CounterGo/common"
	eval "github.com/ChizhovVadim/CounterGo/eval/counter"
	"golang.org/x/sync/errgroup"
)

type IQuietService interface {
	IsQuiet(p *common.Position) bool
}

func QuietServiceBuilder() IQuietService {
	return NewQuietService(eval.NewEvaluationService(), 40)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var err = run()
	if err != nil {
		log.Println(err)
	}
}

type Settings struct {
	GamesFolder string
	ResultPath  string
	Threads     int
}

func run() error {
	curUser, err := user.Current()
	if err != nil {
		return err
	}
	homeDir := curUser.HomeDir
	if homeDir == "" {
		return fmt.Errorf("current user home dir empty")
	}

	var chessDir = filepath.Join(homeDir, "chess")

	var settings = Settings{
		GamesFolder: filepath.Join(chessDir, "pgn"),
		ResultPath:  filepath.Join(chessDir, "fengen.txt"),
		Threads:     max(1, runtime.NumCPU()/2),
	}

	flag.StringVar(&settings.GamesFolder, "input", settings.GamesFolder, "Path to folder with PGN files")
	flag.StringVar(&settings.ResultPath, "output", settings.ResultPath, "Path to output fen file")
	flag.IntVar(&settings.Threads, "threads", settings.Threads, "Number of threads")
	flag.Parse()

	log.Printf("%+v", settings)

	pgnFiles, err := pgnFiles(settings.GamesFolder)
	if err != nil {
		return err
	}
	if len(pgnFiles) == 0 {
		return fmt.Errorf("At least one PGN file is expected")
	}

	return fengenPipeline(context.Background(), QuietServiceBuilder, settings.Threads, pgnFiles, settings.ResultPath)
}

func fengenPipeline(
	ctx context.Context,
	quietServiceBuilder func() IQuietService, //for each thread
	threads int,
	pgnFiles []string,
	resultPath string,
) error {

	log.Println("fengen started")
	defer log.Println("fengen finished")

	g, ctx := errgroup.WithContext(ctx)

	var pgns = make(chan string, 128)
	var games = make(chan Game, 128)

	g.Go(func() error {
		defer close(pgns)
		return loadPgnsManyFiles(ctx, pgnFiles, pgns)
	})

	g.Go(func() error {
		return saveFens(ctx, games, resultPath)
	})

	var wg = &sync.WaitGroup{}

	for i := 0; i < threads; i++ {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			return analyzeGames(ctx, quietServiceBuilder(), pgns, games)
		})
	}

	g.Go(func() error {
		wg.Wait()
		close(games)
		return nil
	})

	return g.Wait()
}

func pgnFiles(folderPath string) ([]string, error) {
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".pgn" {
			result = append(result, filepath.Join(folderPath, file.Name()))
		}
	}
	return result, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
