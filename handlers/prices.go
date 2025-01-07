package handlers

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
    "bytes"
)

// PostPrices обрабатывает загрузку ZIP-файла и сохранение данных в базу
func PostPrices(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		// Извлечение файла из запроса
		file, _, err := r.FormFile("file")
		if err != nil {
			log.Printf("Error retrieving file: %v", err)
			http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Создание временного файла
		tempFile, err := os.CreateTemp("", "uploaded-*.zip")
		if err != nil {
			log.Printf("Error creating temp file: %v", err)
			http.Error(w, "Failed to process file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Копирование данных из запроса во временный файл
		_, err = io.Copy(tempFile, file)
		if err != nil {
			log.Printf("Error copying file data: %v", err)
			http.Error(w, "Failed to process file", http.StatusInternalServerError)
			return
		}

		// Открытие ZIP-архива
		archive, err := zip.OpenReader(tempFile.Name())
		if err != nil {
			log.Printf("Error opening zip file: %v", err)
			http.Error(w, "Failed to open zip file", http.StatusBadRequest)
			return
		}
		defer archive.Close()

		var totalItems int
		var totalCategories int
		var totalPrice float64
		categoriesSet := make(map[string]struct{})

		// Обработка содержимого ZIP-архива
		for _, file := range archive.File {
			if filepath.Ext(file.Name) != ".csv" {
				log.Printf("Skipping non-CSV file: %s", file.Name)
				continue
			}

			// Открываем CSV-файл
			f, err := file.Open()
			if err != nil {
				log.Printf("Error opening file in zip: %v", err)
				continue
			}
			defer f.Close()

			// Читаем CSV-данные
			reader := csv.NewReader(f)

			// Пропускаем заголовок
			header, err := reader.Read()
			if err != nil {
				log.Printf("Error reading CSV header: %v", err)
				continue
			}

			if len(header) != 5 || header[0] != "id" || header[1] != "name" || header[2] != "category" || header[3] != "price" || header[4] != "create_date" {
				log.Printf("Invalid CSV format: %v", header)
				http.Error(w, "Invalid CSV format", http.StatusBadRequest)
				return
			}

			// Обрабатываем строки CSV
			for {
				record, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("Error reading CSV file: %v", err)
					continue
				}

				// Парсинг строки
				id, err := strconv.Atoi(record[0])
				if err != nil {
					log.Printf("Invalid ID format: %v", record[0])
					continue
				}
				name := record[1]
				category := record[2]
				price, err := strconv.ParseFloat(record[3], 64)
				if err != nil {
					log.Printf("Invalid price format: %v", record[3])
					continue
				}
				createDate, err := time.Parse("2006-01-02", record[4])
				if err != nil {
					log.Printf("Invalid date format: %v", record[4])
					continue
				}

				// Добавляем категорию в множество
				categoriesSet[category] = struct{}{}

				// Сохраняем данные в базу
				_, err = db.Exec(
					`INSERT INTO prices (id, created_at, name, category, price) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`,
					id, createDate, name, category, price,
				)
				if err != nil {
					log.Printf("Error inserting data into database: %v", err)
					continue
				}

				totalItems++
				totalPrice += price
			}
		}

		totalCategories = len(categoriesSet)

		// Формируем ответ
		response := map[string]interface{}{
			"total_items":      totalItems,
			"total_categories": totalCategories,
			"total_price":      totalPrice,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
	}
}

// GetPrices возвращает ZIP-архив с данными из базы
func GetPrices(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем данные из базы
		rows, err := db.Query(`SELECT id, created_at, name, category, price FROM prices`)
		if err != nil {
			http.Error(w, "Failed to retrieve data", http.StatusInternalServerError)
			log.Printf("DB error: %v", err)
			return
		}
		defer rows.Close()

		// Буфер для CSV
		csvBuffer := &bytes.Buffer{}
		writer := csv.NewWriter(csvBuffer)
		if err := writer.Write([]string{"id", "name", "category", "price", "create_date"}); err != nil {
			http.Error(w, "Failed to write CSV", http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			var id int
			var createdAt time.Time
			var name, category string
			var price float64
			if err := rows.Scan(&id, &createdAt, &name, &category, &price); err != nil {
				log.Printf("Row scan error: %v", err)
				continue
			}
			writer.Write([]string{
				strconv.Itoa(id),
				name,
				category,
				strconv.FormatFloat(price, 'f', 2, 64),
				createdAt.Format("2006-01-02"),
			})
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			http.Error(w, "Failed to write CSV", http.StatusInternalServerError)
			return
		}

		// Создание ZIP-архива
		zipBuffer := &bytes.Buffer{}
		zipWriter := zip.NewWriter(zipBuffer)
		csvFile, err := zipWriter.Create("data.csv")
		if err != nil {
			http.Error(w, "Failed to create ZIP", http.StatusInternalServerError)
			log.Printf("ZIP error: %v", err)
			return
		}
		if _, err := csvFile.Write(csvBuffer.Bytes()); err != nil {
			http.Error(w, "Failed to write ZIP", http.StatusInternalServerError)
			return
		}
		zipWriter.Close()

		// Отправка файла
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=data.zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipBuffer.Bytes())
	}
}
