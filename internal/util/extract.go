package util

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/gen2brain/go-fitz"
)

// ExtractPDFOCR ekstrak teks dari PDF menggunakan OCR (Tesseract)
func ExtractPDFOCR(path string) (string, error) {
	// Cek apakah tesseract terinstall
	log.Println("Checking tesseract...")
	if err := checkTesseract(); err != nil {
		return "", fmt.Errorf("tesseract check failed: %w", err)
	}

	doc, err := fitz.New(path)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	log.Printf("Total pages: %d\n", doc.NumPage())

	var fullText bytes.Buffer
	var lastErr error

	for n := 0; n < doc.NumPage(); n++ {
		log.Printf("Processing page %d...\n", n+1)

		// Ekstrak gambar dengan resolusi lebih tinggi
		img, err := doc.Image(n)
		if err != nil {
			lastErr = fmt.Errorf("page %d: failed to extract image: %w", n+1, err)
			log.Println(lastErr)
			continue
		}

		// Buat temporary file di sistem temp folder
		tmpFile, err := os.CreateTemp("", "page-*.png")
		if err != nil {
			lastErr = fmt.Errorf("page %d: failed to create temp file: %w", n+1, err)
			log.Println(lastErr)
			continue
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath)

		// Simpan gambar
		err = savePNG(tmpPath, img)
		if err != nil {
			lastErr = fmt.Errorf("page %d: failed to save PNG: %w", n+1, err)
			log.Println(lastErr)
			continue
		}

		// Jalankan Tesseract dengan error handling yang lebih baik
		cmd := exec.Command("tesseract", tmpPath, "stdout", "-l", "eng")
		out, err := cmd.CombinedOutput()

		if err != nil {
			lastErr = fmt.Errorf("page %d: tesseract error: %w, output: %s", n+1, err, string(out))
			log.Println(lastErr)
			continue
		}

		pageText := strings.TrimSpace(string(out))
		log.Printf("Page %d OCR output length: %d chars\n", n+1, len(pageText))

		if len(pageText) > 0 {
			fullText.WriteString(pageText)
			fullText.WriteString("\n\n")
		}
	}

	result := strings.TrimSpace(fullText.String())

	if len(result) == 0 {
		if lastErr != nil {
			return "", fmt.Errorf("failed to extract text via OCR: %w", lastErr)
		}
		return "", fmt.Errorf("no text extracted from PDF (PDF might be empty or images are unreadable)")
	} else if len(result) < 100 {
		return "", fmt.Errorf("content too short for meaningful evaluation")
	}

	log.Printf("Total extracted text: %d chars\n", len(result))
	return result, nil
}

// checkTesseract memverifikasi apakah tesseract terinstall dan bisa dijalankan
func checkTesseract() error {
	cmd := exec.Command("tesseract", "-v")
	out, err := cmd.CombinedOutput()
	log.Println("Tesseract Output:", string(out))
	if err != nil {
		return fmt.Errorf("tesseract not found or not executable: %w\nOutput: %s", err, string(out))
	}
	log.Printf("Tesseract version: %s\n", strings.Split(string(out), "\n")[0])
	return nil
}

func savePNG(path string, img interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Konversi interface{} ke image.Image
	i, ok := img.(image.Image)
	if !ok {
		return fmt.Errorf("invalid image type: %T", img)
	}

	if err := png.Encode(f, i); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}
