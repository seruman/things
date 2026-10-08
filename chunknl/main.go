package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dustin/go-humanize"
)

func main() {
	if err := realMain(
		os.Stdout,
		os.Stderr,
		os.Args,
	); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(stdout io.Writer, stderr io.Writer, osargs []string) error {
	fs := flag.NewFlagSet("chunknl", flag.ExitOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [options] <filepath>\n", fs.Name())
		fs.PrintDefaults()
	}

	flagLinesPerChunk := fs.Int("l", 10_00_000, "lines per chunk")
	flagSize := fs.Bool("s", false, "show size in bytes")
	flagHumanize := fs.Bool("h", false, "show size in human-readable format")
	flagShowCount := fs.Bool("n", false, "show line count")
	flagBufferSizeMB := fs.Int("b", 50, "buffer size in MB")

	if err := fs.Parse(osargs[1:]); err != nil {
		return err
	}

	args := fs.Args()

	if len(args) != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one filepath argument")
	}

	if *flagLinesPerChunk <= 0 {
		return fmt.Errorf("lines per chunk (-l) must be positive")
	}

	filepath := args[0]

	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	const MiB = 1024 * 1024
	bufferSize := *flagBufferSizeMB * MiB
	reader := bufio.NewReaderSize(file, bufferSize)

	var currentOffset int64
	var chunkStartOffset int64
	var lineCount int
	var totalChunks int
	var lineSize int64

	for {
		fragment, err := reader.ReadSlice('\n')
		lineSize += int64(len(fragment))
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil && err != io.EOF {
			return err
		}

		if lineSize > 0 {
			lineCount++
			currentOffset += lineSize
			lineSize = 0

			if lineCount == *flagLinesPerChunk {
				totalChunks++
				size := currentOffset - chunkStartOffset

				fmt.Printf("%d", chunkStartOffset)

				if *flagSize {
					sizeStr := fmt.Sprintf("%d", size)
					if *flagHumanize {
						sizeStr = humanize.Bytes(uint64(size))
					}
					fmt.Printf(" %s", sizeStr)
				}

				if *flagShowCount {
					fmt.Printf(" %d lines", lineCount)
				}
				fmt.Println()

				chunkStartOffset = currentOffset
				lineCount = 0
			}
		}

		if err == io.EOF {
			break
		}
	}

	// Report last chunk if there are remaining lines
	if lineCount > 0 {
		totalChunks++
		size := currentOffset - chunkStartOffset

		fmt.Printf("%d", chunkStartOffset)

		if *flagSize {
			sizeStr := fmt.Sprintf("%d", size)
			if *flagHumanize {
				sizeStr = humanize.Bytes(uint64(size))
			}
			fmt.Printf(" %s", sizeStr)
		}

		if *flagShowCount {
			fmt.Printf(" %d lines", lineCount)
		}
		fmt.Println()
	}

	fmt.Fprintf(stderr, "Total chunks: %d\n", totalChunks)

	return nil
}
