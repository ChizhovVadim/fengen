# FenGen

A simple utility to extract FENs from PGN databases.
This utility is useful for extracting FENs from PGN databases to be used for
tuning Hand-Crafted Evaluation terms, as well as training NNUE networks.

Project idea taken from Zahak:
+ [Zahak training](https://github.com/amanjpro/zahak/blob/master/training.md)
+ [Zahak fengen](https://github.com/amanjpro/fengen)

## Usage

```
$ ./fengen -help
Usage of ./fengen:
  -input string
        Path to folder with PGN files (default "/Users/vadimchizhov/chess/pgn")
  -output string
        Path to output fen file (default "/Users/vadimchizhov/chess/fengen.txt")
  -threads int
        Number of threads (default 4)
```
