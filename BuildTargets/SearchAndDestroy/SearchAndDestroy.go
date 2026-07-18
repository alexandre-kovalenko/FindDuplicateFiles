package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"time"

	"rabbitsden.online/FindDuplicateFiles/Constants"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Takes regular expression and the file name, finds all lines in the file that match the regular expression,
// interprets them as the filenames and deletes files with these names.
// Basically, this is an equivalent of
// 	 egrep <regular expression> <file> | xargs rm
// but works on the platforms where neither grep nor xargs are available (Windows, I am looking at you) and
// also deals with filenames with the special characters correctly.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	if len(os.Args) != 3 {
		log.Printf("Usage: %s <regular expression> <file>\n", os.Args[0])
		os.Exit(1)
	}
	startTS := time.Now()
	nFilesDeleted, nFilesNotDeleted, err := findAndDeleteFiles(os.Args[1], os.Args[2])
	if err != nil {
		log.Printf("Error %s after deleting %d files and skipping %d files\n",
			err, nFilesDeleted, nFilesNotDeleted)
		os.Exit(1)
	}
	endTS := time.Now()
	log.Printf("Successfully deleted files %d and skipped %d files in %d ms (%.02f files/s) \n",
		nFilesDeleted, nFilesNotDeleted,
		endTS.Sub(startTS)/time.Millisecond, float64(nFilesDeleted)/(float64(endTS.Sub(startTS))/float64(time.Second)))
}

const (
	STATE_START        = 0
	STATE_IN_THE_BLOCK = 1
)

func processBlock(filesToRemove []string, nToBeKept int) (int, int, error) {
	nDeleted := 0
	nNotDeleted := 0
	for _, fname := range filesToRemove {
		if nToBeKept == 0 {
			log.Printf("Not deleting %s because there will be no such files left\n", fname)
			nNotDeleted++
		} else {
			log.Printf("Deleting %s\n", fname)
			if err := os.Remove(fname); err != nil {
				return nDeleted, nNotDeleted, err
			}
			nDeleted++
		}
	}
	return nDeleted, nNotDeleted, nil
}

func findAndDeleteFiles(pattern string, fileName string) (int, int, error) {
	rFile := regexp.MustCompile(pattern)
	rSeparator := regexp.MustCompile(`^` + Constants.BLOCK_SEPARATOR + `$`)
	file, err := os.Open(fileName)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	nDeleted := 0
	nNotDeleted := 0
	scanner := bufio.NewScanner(file)
	state := STATE_START
	nToBeKept := 0
	var filesToRemove []string
	for scanner.Scan() {
		line := scanner.Text()
		if rSeparator.MatchString(line) {
			if state == STATE_IN_THE_BLOCK {
				d, nd, err := processBlock(filesToRemove, nToBeKept)
				nDeleted += d
				nNotDeleted += nd
				if err != nil {
					return nDeleted, nNotDeleted, err
				}
			}
			state = STATE_IN_THE_BLOCK
			filesToRemove = make([]string, 0)
			nToBeKept = 0
			continue
		}
		if state == STATE_IN_THE_BLOCK {
			if rFile.MatchString(line) {
				filesToRemove = append(filesToRemove, line)
			} else {
				nToBeKept++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nDeleted, nNotDeleted, err
	}
	if state == STATE_IN_THE_BLOCK {
		d, nd, err := processBlock(filesToRemove, nToBeKept)
		nDeleted += d
		nNotDeleted += nd
		if err != nil {
			return nDeleted, nNotDeleted, err
		}
	}
	return nDeleted, nNotDeleted, nil
}
