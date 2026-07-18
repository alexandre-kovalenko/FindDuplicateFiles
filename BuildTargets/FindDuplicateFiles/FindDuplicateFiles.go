package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"rabbitsden.online/FindDuplicateFiles/Constants"
	"rabbitsden.online/FindDuplicateFiles/DirectoryUnit"
	"rabbitsden.online/FindDuplicateFiles/Helper"
	"rabbitsden.online/FindDuplicateFiles/UnitLimiter"
)

func Usage() {
	log.Printf("Usage: %s [-o <result filename>] <directory> [<directory>...]\n", os.Args[0])
	os.Exit(-1)
}

func main() {
	if len(os.Args) < 2 {
		Usage()
	}
	if (os.Args[1] == "-o") && (len(os.Args) < 4) {
		Usage()
	}
	var resultFilename string
	firstDirectoryIndex := 1
	if os.Args[1] == "-o" {
		resultFilename = os.Args[2]
		_ = os.Remove(resultFilename)
		firstDirectoryIndex = 3
	}
	for i := firstDirectoryIndex; i < len(os.Args); i++ {
		if err := Helper.ValidateAbsolutePath(filepath.Clean(os.Args[i])); err != nil {
			log.Printf("Invalid path: %v\n", err)
			os.Exit(-1)
		}
	}
	// Since we are heavily I/O bound, let's schedule 8 goroutines per core
	runtime.GOMAXPROCS(runtime.NumCPU() * 8)
	log.Printf("Running on %d cores, max procs is: %d\n", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	startTS := time.Now()
	fl := UnitLimiter.MakeUnitLimiter(1000, "file")
	csMap := make(map[string][]string)
	filesProcessed := 0
	for i := firstDirectoryIndex; i < len(os.Args); i++ {
		rootDirectory := os.Args[i]
		mdu := DirectoryUnit.MakeDirectoryUnits(rootDirectory, &fl)
		dUnits, err := mdu.DirectoryUnits, mdu.Error
		if err != nil {
			log.Printf("Failed to enumerate %s (%v)\n", rootDirectory, err)
			os.Exit(-1)
		}
		for _, du := range dUnits {
			for _, f := range du.PlainFiles {
				eSum := f.GetEncodedChecksum()
				filesProcessed++
				csMap[eSum] = append(csMap[eSum], f.Name)
			}
		}
	}
	for _, v := range csMap {
		if len(v) > 1 {
			err := processDuplicates(v, resultFilename)
			if err != nil {
				log.Printf("Failed to process duplicates in %v -- (%v)\n", v, err)
				os.Exit(-1)
			}
		}
	}
	endTS := time.Now()
	elapsedMS := endTS.Sub(startTS).Milliseconds()
	rate := 0.0
	if elapsedMS > 0 {
		rate = float64(filesProcessed) / (float64(elapsedMS) / 1000)
	}
	log.Printf("Running on %d cores, max procs is: %d\n", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	log.Printf("Processed %d files in %d ms (%.2f files/s)\n", filesProcessed, elapsedMS, rate)
}

// //////////////////////////////////////////////////////////////////////////////////
// Process duplicate files

func processDuplicates(duplicates []string, resultFilename string) error {
	separateResultFile := false
	var fh *os.File
	var err error
	if len(resultFilename) > 0 {
		fh, err = os.OpenFile(resultFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		separateResultFile = true
		fmt.Fprintln(fh, Constants.BLOCK_SEPARATOR)
	}
	defer func() {
		if separateResultFile {
			fh.Close()
		}
	}()
	for _, f := range duplicates {
		log.Println(f)
		if separateResultFile {
			fmt.Fprintf(fh, "%s\n", f)
		}
	}
	return nil
}
