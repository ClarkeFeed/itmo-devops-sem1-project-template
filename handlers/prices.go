package handlers

import (
    "archive/zip"
    "bytes"
    "database/sql"
    "encoding/csv"
    "encoding/json"
    "io"
    "log"
    "math"
    "net/http"
    "path/filepath"
    "strconv"
    "time"
)

// PostResponse описывает структуру JSON-ответа для POST-запроса
type PostResponse struct {
    TotalItems      int     `json:"total_items"`
    TotalCategories int     `json:"total_categories"`
    TotalPrice      float64 `json:"total_price"`
}

// PostPrices обрабатывает загрузку ZIP-архива и запись данных в БД
func PostPrices(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        // Извлекаем файл из формы (поле "file")
        file, _, err := r.FormFile("file")
        if err != nil {
            log.Printf("Error retrieving file: %v", err)
            http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
            return
        }
        defer file.Close()

        // Считываем ZIP-архив в буфер
        zipBuffer := &bytes.Buffer{}
        if _, err := io.Copy(zipBuffer, file); err != nil {
            log.Printf("Error reading uploaded file: %v", err)
            http.Error(w, "Failed to read file", http.StatusInternalServerError)
            return
        }

        // Открываем ZIP-архив из буфера
        zipReader, err := zip.NewReader(bytes.NewReader(zipBuffer.Bytes()), int64(zipBuffer.Len()))
        if err != nil {
            log.Printf("Error opening zip archive: %v", err)
            http.Error(w, "Invalid zip file", http.StatusBadRequest)
            return
        }

        var totalItems int
        var totalPrice float64
        categoriesSet := make(map[string]struct{})

        // Ищем в архиве файл data.csv
        for _, zf := range zipReader.File {
            if filepath.Base(zf.Name) != "data.csv" {
                continue
            }

            csvFile, err := zf.Open()
            if err != nil {
                log.Printf("Error opening CSV file in zip: %v", err)
                continue
            }
            defer csvFile.Close()

            reader := csv.NewReader(csvFile)

            // Явно пропускаем строку заголовка (id,name,category,price,create_date)
            if _, err := reader.Read(); err != nil {
                log.Printf("Error reading CSV header: %v", err)
                continue
            }

            // Читаем все строки
            for {
                record, err := reader.Read()
                if err == io.EOF {
                    break
                }
                if err != nil {
                    log.Printf("Error reading CSV record: %v", err)
                    continue
                }
                if len(record) < 5 {
                    log.Printf("Invalid record: %v", record)
                    continue
                }

                idStr := record[0]
                name := record[1]
                category := record[2]
                priceStr := record[3]
                dateStr := record[4]

                // Парсим цену
                priceVal, err := strconv.ParseFloat(priceStr, 64)
                if err != nil {
                    log.Printf("Invalid price format: %v", priceStr)
                    continue
                }
                // Парсим дату
                createdAt, err := time.Parse("2006-01-02", dateStr)
                if err != nil {
                    log.Printf("Invalid date format: %v", dateStr)
                    continue
                }

                // Вставляем запись в БД (игнорируем дубликаты по id)
                _, err = db.Exec(`INSERT INTO prices (id, created_at, name, category, price)
                                VALUES ($1, $2, $3, $4, $5)
                                ON CONFLICT (id) DO NOTHING`, 
                                idStr, createdAt, name, category, priceVal)
                if err != nil {
                    log.Printf("DB insert error: %v", err)
                    continue
                }

                // Подсчитываем статистику
                totalItems++
                totalPrice += priceVal
                categoriesSet[category] = struct{}{}
            }
        }

        // Формируем JSON-ответ
        resp := PostResponse{
            TotalItems:      totalItems,
            TotalCategories: len(categoriesSet),
            // Округляем суммарную цену до 2 знаков
            TotalPrice: math.Round(totalPrice*100) / 100,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }
}

// GetPrices выгружает все записи из БД в файл data.csv и возвращает его в ZIP-архиве
func GetPrices(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        rows, err := db.Query(`SELECT id, created_at, name, category, price FROM prices`)
        if err != nil {
            log.Printf("Error querying database: %v", err)
            http.Error(w, "Failed to retrieve data", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        // Собираем CSV-данные в буфер
        csvBuffer := &bytes.Buffer{}
        writer := csv.NewWriter(csvBuffer)

        // Пишем заголовок CSV
        writer.Write([]string{"id", "name", "category", "price", "create_date"})

        for rows.Next() {
            var (
                id        int
                createdAt time.Time
                name      string
                category  string
                priceVal  float64
            )

            if err := rows.Scan(&id, &createdAt, &name, &category, &priceVal); err != nil {
                log.Printf("Row scan error: %v", err)
                continue
            }

            writer.Write([]string{
                strconv.Itoa(id),
                name,
                category,
                strconv.FormatFloat(priceVal, 'f', 2, 64),
                createdAt.Format("2006-01-02"),
            })
        }
        writer.Flush()
        if err := writer.Error(); err != nil {
            log.Printf("Error finalizing CSV: %v", err)
            http.Error(w, "Failed to write CSV", http.StatusInternalServerError)
            return
        }

        // Упаковываем CSV-файл в ZIP-архив (в памяти)
        zipBuffer := &bytes.Buffer{}
        zipWriter := zip.NewWriter(zipBuffer)

        csvFile, err := zipWriter.Create("data.csv")
        if err != nil {
            log.Printf("Error creating file in ZIP: %v", err)
            http.Error(w, "Failed to create ZIP", http.StatusInternalServerError)
            return
        }

        if _, err := csvFile.Write(csvBuffer.Bytes()); err != nil {
            log.Printf("Error writing CSV to ZIP: %v", err)
            http.Error(w, "Failed to write ZIP", http.StatusInternalServerError)
            return
        }

        if err := zipWriter.Close(); err != nil {
            log.Printf("Error closing ZIP writer: %v", err)
            http.Error(w, "Failed to close ZIP", http.StatusInternalServerError)
            return
        }

        // Отправляем ZIP-архив клиенту
        w.Header().Set("Content-Type", "application/zip")
        w.Header().Set("Content-Disposition", "attachment; filename=data.zip")
        w.WriteHeader(http.StatusOK)
        if _, err := w.Write(zipBuffer.Bytes()); err != nil {
            log.Printf("Error sending ZIP file: %v", err)
        }
    }
}
