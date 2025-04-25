package main

import (
	"archive/zip"
	"compress/flate"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/beevik/etree"
)

var (
	r2023b = flag.Bool("r2023b", false, "Set output to R2023b")
	r2024a = flag.Bool("r2024a", false, "Set output to R2024a")
	r2024b = flag.Bool("r2024b", false, "Set output to R2024b")
	r2023a = flag.Bool("r2023a", false, "Set output to R2023a")
	r2022b = flag.Bool("r2022b", false, "Set output to R2022b")
	r2022a = flag.Bool("r2022a", false, "Set output to R2022a")
)
var selectedRelease string

func updateVersions(xmlPath string, updates map[string]string) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(xmlPath); err != nil {
		return err
	}
	modified := false
	for tag, val := range updates {
		for _, el := range doc.FindElements("//" + tag) {
			if el.Text() != val {
				el.SetText(val)
				modified = true
			}
		}
	}
	if modified {
		return doc.WriteToFile(xmlPath)
	}
	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		out, err := os.Create(fpath)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, rc); err != nil {
			return err
		}
	}
	return nil
}

func zipDir(src, dest string) error {
	zf, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer zf.Close()

	// Create a new zip writer
	zw := zip.NewWriter(zf)
	defer zw.Close()

	// Use standard Deflate compression
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.DefaultCompression)
	})

	// Don't use UTF-8 flag for file names
	zw.SetComment("") // Empty comment to avoid UTF-8 flag

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Convert Windows backslashes to forward slashes
		rel = strings.ReplaceAll(rel, "\\", "/")

		// Create file header without UTF-8 flag
		header := &zip.FileHeader{
			Name:     rel,
			Method:   zip.Deflate,
			Modified: info.ModTime(),
		}

		// Clear UTF-8 flag - crucial for MATLAB compatibility
		header.Flags &= ^uint16(1 << 11)

		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})

	return err
}

func convertSLX(slx string) (string, error) {
	base := strings.TrimSuffix(slx, filepath.Ext(slx))
	outSLX := base + filepath.Ext(slx)

	workDir := base + "_unzipped"

	os.RemoveAll(workDir)
	if err := os.MkdirAll(workDir, os.ModePerm); err != nil {
		return "", err
	}
	if err := unzip(slx, workDir); err != nil {
		return "", err
	}

	// dynamically apply the chosen release
	updates := map[string]string{
		"version":       selectedRelease,
		"release":       selectedRelease,
		"matlabRelease": selectedRelease,
	}

	xmlFiles := []string{
		filepath.Join(workDir, "metadata", "mwcoreProperties.xml"),
		filepath.Join(workDir, "metadata", "mwcorePropertiesReleaseInfo.xml"),
		filepath.Join(workDir, "metadata", "coreProperties.xml"),
	}
	for _, xf := range xmlFiles {
		if _, err := os.Stat(xf); err == nil {
			if err := updateVersions(xf, updates); err != nil {
				return "", err
			}
		}
	}

	if err := zipDir(workDir, outSLX); err != nil {
		return "", err
	}
	// clean up temporary folder
	os.RemoveAll(workDir)
	return outSLX, nil
}

func processDirectory(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		path := filepath.Join(dir, file.Name())

		if file.IsDir() {
			// Recursively process subdirectories
			if err := processDirectory(path); err != nil {
				return err
			}
		} else {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext == ".slx" || ext == ".sldd" || ext == ".mldatx" {
				// Process SLX, SLDD, or MLDATX file
				fmt.Printf("Processing: %s\n", path)
				out, err := convertSLX(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", path, err)
					continue // Continue with next file on error
				}
				fmt.Println("Created:", out)
			}
		}
	}
	return nil
}

func main() {
	// Define command-line flags
	recursiveFlag := flag.Bool("d", false, "Process directory recursively")
	recursiveLongFlag := flag.Bool("directory", false, "Process directory recursively")

	// Custom usage message
	flag.Usage = func() {
		prog := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input.slx|.sldd|.mldatx or directory>\n\n", prog)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -d, --directory    Process all .slx/.sldd/.mldatx files in directory recursively\n")
		fmt.Fprintf(os.Stderr, "  --r2022a           Set output to R2023b\n")
		fmt.Fprintf(os.Stderr, "  --r2022b           Set output to R2024a\n")
		fmt.Fprintf(os.Stderr, "  --r2023a           Set output to R2024b\n")
		fmt.Fprintf(os.Stderr, "  --r2023b           Set output to R2023a\n")
		fmt.Fprintf(os.Stderr, "  --r2024a           Set output to R2022b\n")
		fmt.Fprintf(os.Stderr, "  --r2024b           Set output to R2022a\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s model.slx                  # Convert a single file\n", prog)
		fmt.Fprintf(os.Stderr, "  %s data.sldd                  # Convert a single file\n", prog)
		fmt.Fprintf(os.Stderr, "  %s -d folder_with_archives    # Convert all .slx, .sldd, or .mldatx files in directory\n", prog)
	}

	flag.Parse()

	// ensure exactly one release flag is set
	count := 0
	if *r2023b {
		count++
		selectedRelease = "R2023b"
	}
	if *r2024a {
		count++
		selectedRelease = "R2024a"
	}
	if *r2024b {
		count++
		selectedRelease = "R2024b"
	}
	if *r2023a {
		count++
		selectedRelease = "R2023a"
	}
	if *r2022b {
		count++
		selectedRelease = "R2022b"
	}
	if *r2022a {
		count++
		selectedRelease = "R2022a"
	}
	if count != 1 {
		fmt.Fprintln(os.Stderr, "Error: must specify exactly one of --r2022a, --r2022b, --r2023a, --r2023b, --r2024a, or --r2024b")
		flag.Usage()
		os.Exit(1)
	}

	// Check arguments
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Get the path argument
	path := args[0]
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Determine if recursive mode is enabled (either flag will work)
	recursiveMode := *recursiveFlag || *recursiveLongFlag

	if fileInfo.IsDir() {
		if recursiveMode {
			// Process all SLX files in directory recursively
			if err := processDirectory(path); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s is a directory. Use -r or --recursive to process directories.\n", path)
			flag.Usage()
			os.Exit(1)
		}
	} else {
		// Process single file
		out, err := convertSLX(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		fmt.Println("Created:", out)
	}
}
