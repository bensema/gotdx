package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	meta, err := client.GetFileMeta(gotdx.BlockFileDefault)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("file_meta name=%s size=%d", gotdx.BlockFileDefault, meta.Size)

	chunk, err := client.DownloadFile(gotdx.BlockFileDefault, 0, 1024)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("download_chunk size=%d", chunk.Size)

	full, err := client.DownloadFullFile(gotdx.BlockFileDefault, meta.Size)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("download_full size=%d", len(full))

	flat, err := client.GetParsedBlockFile(gotdx.BlockFileGN)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range flat[:min(10, len(flat))] {
		log.Printf("flat block=%s type=%d code=%s", item.BlockName, item.BlockType, item.Code)
	}

	grouped, err := client.GetGroupedBlockFile(gotdx.BlockFileFG)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range grouped[:min(5, len(grouped))] {
		log.Printf("grouped block=%s type=%d stocks=%d", item.BlockName, item.BlockType, item.StockCount)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
