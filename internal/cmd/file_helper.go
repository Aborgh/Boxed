package cmd

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
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

func SaveFileAndComputeChecksums(fileHeader *multipart.FileHeader, destinationPath string) (sha256sum string, sha512sum string, err error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	dst, err := os.Create(destinationPath)
	if err != nil {
		return "", "", err
	}
	defer dst.Close()

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
