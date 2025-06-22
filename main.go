package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <zip|tar file>")
		return
	}

	inputFile := os.Args[1]
	var destDir string
	var err error

	switch filepath.Ext(inputFile) {
	case ".zip":
		destDir, err = unzip(inputFile)
		if err != nil {
			fmt.Printf("unzip error: %v\n", err)
			return
		}
	case ".tar":
		destDir, err = untar(inputFile)
		if err != nil {
			fmt.Printf("untar error: %v\n", err)
			return
		}
	default:
		fmt.Println("Unsupported file type. Only .zip and .tar supported.")
		return
	}

	if err != nil {
		fmt.Printf("Extraction error: %v\n", err)
		return
	}

	fmt.Printf("Extracted directory: %v\n", destDir)

	defer os.RemoveAll(destDir)

	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		fmt.Printf("Destination directory does not exist: %v\n", destDir)
		return
	}

	err = filepath.Walk(destDir, processFile)
	if err != nil {
		fmt.Printf("Traversal error: %v\n", err)
	}
}

func unzip(src string) (string, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer r.Close()

	dest, err := os.MkdirTemp("", "logs_unzip_")
	if err != nil {
		return "", err
	}

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", err
		}

		dstFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return "", err
		}

		if _, err = io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return "", err
		}

		srcFile.Close()
		dstFile.Close()
	}

	return dest, nil
}

func untar(src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()

	dest, err := os.MkdirTemp("", "logs_untar_")
	if err != nil {
		return "", err
	}

	tr := tar.NewReader(f)

	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				return "", err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
		}
	}

	return dest, nil
}

func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".log" || ext == ".txt" {
		return parseFile(path)
	}

	return nil
}

func parseFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {
		text := scanner.Text()
		lowerText := strings.ToLower(text)

		if strings.Contains(lowerText, "error") || strings.Contains(lowerText, "timeout") {
			fmt.Printf("[%s:%d]: %s\n", path, lineNum, text)
		}
		lineNum++
	}

	return scanner.Err()
}
