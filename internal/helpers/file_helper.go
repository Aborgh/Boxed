package helpers

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

func GetFileType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext != "" {
		return ext[1:]
	}
	return "unknown"
}

func SaveFileAndComputeChecksums(fileHeader *multipart.FileHeader, destinationPath string) (
	sha256sum string,
	sha512sum string,
	err error,
) {
	src, err := fileHeader.Open()

	if err != nil {
		return "", "", err
	}

	defer func(src multipart.File) {
		err = src.Close()
		if err != nil {
			return
		}
	}(src)
	if err != nil {
		return "", "", err
	}

	dst, err := os.Create(destinationPath)
	if err != nil {
		return "", "", err
	}
	defer func(dst *os.File) {
		err = dst.Close()
		if err != nil {
			return
		}
	}(dst)

	if err != nil {
		return "", "", err
	}

	sha256Hasher := sha256.New()
	sha512Hasher := sha512.New()

	writer := io.MultiWriter(dst, sha256Hasher, sha512Hasher)

	if _, err := io.Copy(writer, src); err != nil {
		return "", "", err
	}

	sha256sum = hex.EncodeToString(sha256Hasher.Sum(nil))
	sha512sum = hex.EncodeToString(sha512Hasher.Sum(nil))

	return sha256sum, sha512sum, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	var deferErr error
	if err != nil {
		return fmt.Errorf("could not open source file: %w", err)
	}
	defer func(sourceFile *os.File) {
		err := sourceFile.Close()
		if err != nil {
			deferErr = err
		}
	}(sourceFile)
	if deferErr != nil {
		return deferErr
	}
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create destination file: %w", err)
	}
	defer func(destFile *os.File) {
		err := destFile.Close()
		if err != nil {
			deferErr = err

		}
	}(destFile)
	if deferErr != nil {
		return deferErr
	}
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("could not copy file: %w", err)
	}

	// Ensure file permissions are set properly
	return os.Chmod(dst, 0644)
}

// ComputeChecksums calculates SHA256 and SHA512 hashes for a file
func ComputeChecksums(filePath string) (sha256sum string, sha512sum string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("could not open file for checksums: %w", err)
	}
	defer file.Close()

	sha256Hasher := sha256.New()
	sha512Hasher := sha512.New()

	// Create a multi-writer to write to both hashers simultaneously
	multiWriter := io.MultiWriter(sha256Hasher, sha512Hasher)

	if _, err := io.Copy(multiWriter, file); err != nil {
		return "", "", fmt.Errorf("could not compute checksums: %w", err)
	}

	sha256sum = hex.EncodeToString(sha256Hasher.Sum(nil))
	sha512sum = hex.EncodeToString(sha512Hasher.Sum(nil))

	return sha256sum, sha512sum, nil
}

func DeleteFile(path string, recurse bool) error {
	if recurse {
		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
	} else {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidatePath checks if a path uses the correct format (hyphens, not underscores)
func ValidatePath(path string) error {
	if strings.Contains(path, "_") {
		return fmt.Errorf("invalid path format: use hyphens (-) instead of underscores (_)")
	}
	return nil
}
