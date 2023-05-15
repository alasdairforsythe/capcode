package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/alasdairforsythe/capcode/go"
)

func main() {
	// Check if the --help option is present
	if (len(os.Args) > 1 && os.Args[1] == "--help") || len(os.Args) <= 1 {
		printHelp()
		os.Exit(0)
	}

	var from string
	var to string
	var decode bool
	var all bool

	// Check if optional arguments are present
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-d":
			decode = true
		case "-all":
			all = true
		default:
			if arg[0] == '-' {
				fmt.Println("Invalid option:", arg)
				os.Exit(1)
			}
			if len(from) == 0 {
				from = arg
			} else {
				if len(to) == 0 {
					to = arg
				} else {
					fmt.Println("Invalid options")
				}
			}
		}
	}

	if len(from) == 0 {
		printHelp()
		os.Exit(1)
	}

	// Check if conflicting options are present
	if all && decode {
		fmt.Println("Invalid options: -all is available only for encoding, not decoding")
		os.Exit(1)
	}

	// Check if the source file or directory exists
	sourceInfo, err := os.Stat(from)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Source file or directory does not exist:", from)
		} else {
			fmt.Println("Error checking source:", err)
		}
		os.Exit(1)
	}

	// Perform the encoding or decoding based on the flags
	if all {
		if !sourceInfo.IsDir() {
			fmt.Println("Source is not a directory")
			os.Exit(1)
		}
		err := processAllFiles(from, decode)
		if err != nil {
			fmt.Println("Error:", err)
		}
	} else {
		// Check if the source is a directory
		if sourceInfo.IsDir() {
			fmt.Println("Source file is directory, use -all for processing a directory")
			os.Exit(1)
		}
		err := processFile(from, to, decode)
		if err != nil {
			fmt.Println("Error:", err)
		}
	}
}

func printHelp() {
	fmt.Println("Usage: ./capcode <from> <to> [-d] [-all]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  <source>  : Source file or directory (required)")
	fmt.Println("  <dest>    : Destination file or directory (optional)")
	fmt.Println("  -d        : Decode file (default: encode)")
	fmt.Println("  -all      : Encode all files in directory")
	fmt.Println("  --help    : Print this help message")
}

func processFile(from string, to string, decode bool) error {
	// Check if "to" argument is blank and generate a default value
	if to == "" {
		if decode {
			to = from + ".decoded"
		} else {
			to = from + ".toknorm"
		}
	}

	if decode {
		err := capcode.DecodeFile(from, to)
		if err != nil {
			return err
		}
		fmt.Println(`Decoded:`, from)
	} else {
		err := capcode.EncodeFile(from, to)
		if err != nil {
			return err
		}
		fmt.Println(`Encoded:`, from)
	}

	return nil
}

func processAllFiles(directory string, decode bool) error {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			from := filepath.Join(directory, file.Name())
			to := from + ".toknorm"
			if decode {
				to = from + ".decoded"
			}
			err := processFile(from, to, decode)
			if err != nil {
				fmt.Printf("Error processing file '%s': %v\n", from, err)
			}
		}
	}

	return nil
}
