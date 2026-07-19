// Package importer parses bulk target lists from CSV or Excel (.xlsx) files.
// The header row is matched flexibly against common Turkish and English column
// names, so operators can upload a spreadsheet exported from any HR system
// without reformatting it first.
package importer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Row is one parsed target record.
type Row struct {
	Email      string
	FirstName  string
	LastName   string
	Position   string
	Department string
	Timezone   string
	VIP        bool
}

// ParseFile parses a target list. filename picks the parser by extension
// (".xlsx" uses the Excel parser; anything else is treated as CSV/text).
// It returns the successfully parsed rows plus human-readable messages for
// rows that were skipped (e.g. missing/invalid email).
func ParseFile(filename string, data []byte) ([]Row, []string, error) {
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return parseXLSX(data)
	}
	return parseCSV(data)
}

func parseCSV(data []byte) ([]Row, []string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // strip UTF-8 BOM (common from Excel "Save as CSV")
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = detectDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("CSV dosyası ayrıştırılamadı: %w", err)
	}
	return parseRecords(records)
}

// detectDelimiter chooses ',' or ';' based on the header line — Turkish Excel
// locales default to ';' for CSV export since ',' is the decimal separator.
func detectDelimiter(data []byte) rune {
	line := data
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		line = data[:i]
	}
	if bytes.Count(line, []byte{';'}) > bytes.Count(line, []byte{','}) {
		return ';'
	}
	return ','
}

func parseXLSX(data []byte) ([]Row, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("Excel dosyası açılamadı: %w", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, nil, fmt.Errorf("Excel dosyasında sayfa bulunamadı")
	}
	records, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("Excel satırları okunamadı: %w", err)
	}
	return parseRecords(records)
}

// normalizeHeader folds a header cell to a comparable key: lowercased,
// Turkish characters transliterated, and spaces/punctuation stripped —
// so "E-Posta Adresi", "eposta_adresi" and "Email" all resolve consistently.
func normalizeHeader(s string) string {
	replacer := strings.NewReplacer(
		"İ", "i", "I", "i", "ı", "i",
		"Ş", "s", "ş", "s",
		"Ğ", "g", "ğ", "g",
		"Ü", "u", "ü", "u",
		"Ö", "o", "ö", "o",
		"Ç", "c", "ç", "c",
		" ", "", "-", "", "_", "",
	)
	return strings.ToLower(replacer.Replace(strings.TrimSpace(s)))
}

func aliasSet(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

var (
	emailAliases      = aliasSet("email", "eposta", "mail", "epostaadresi", "mailadresi")
	fullNameAliases   = aliasSet("adsoyad", "isim", "fullname", "name", "adisoyadi")
	firstNameAliases  = aliasSet("ad", "firstname", "adi", "isimad")
	lastNameAliases   = aliasSet("soyad", "lastname", "soyadi")
	departmentAliases = aliasSet("departman", "department", "birim", "takim", "ekip")
	positionAliases   = aliasSet("pozisyon", "position", "unvan", "gorev", "title")
	timezoneAliases   = aliasSet("saatdilimi", "zamandilimi", "timezone", "tz")
	vipAliases        = aliasSet("vip", "onemli", "oncelikli")
)

type columns struct {
	email, fullName, firstName, lastName, department, position, timezone, vip int
}

// parseRecords maps a header row to known columns, then extracts a Row per
// data row. Rows with a missing/invalid email are reported and skipped;
// blank rows are silently skipped (common trailing rows in spreadsheets).
func parseRecords(records [][]string) ([]Row, []string, error) {
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("dosya boş")
	}
	cols := columns{-1, -1, -1, -1, -1, -1, -1, -1}
	for i, h := range records[0] {
		switch n := normalizeHeader(h); {
		case emailAliases[n]:
			cols.email = i
		case fullNameAliases[n]:
			cols.fullName = i
		case firstNameAliases[n]:
			cols.firstName = i
		case lastNameAliases[n]:
			cols.lastName = i
		case departmentAliases[n]:
			cols.department = i
		case positionAliases[n]:
			cols.position = i
		case timezoneAliases[n]:
			cols.timezone = i
		case vipAliases[n]:
			cols.vip = i
		}
	}
	if cols.email < 0 {
		return nil, nil, fmt.Errorf(`e-posta sütunu bulunamadı — başlık satırında "E-posta" veya "Email" adında bir sütun olmalı`)
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	var rows []Row
	var errs []string
	for i, row := range records[1:] {
		lineNo := i + 2 // account for the header row, 1-indexed
		email := strings.ToLower(get(row, cols.email))
		if email == "" {
			continue
		}
		if !strings.Contains(email, "@") {
			errs = append(errs, fmt.Sprintf("satır %d: geçersiz e-posta %q", lineNo, email))
			continue
		}
		first, last := get(row, cols.firstName), get(row, cols.lastName)
		if cols.fullName >= 0 {
			if full := get(row, cols.fullName); full != "" {
				if idx := strings.LastIndex(full, " "); idx > 0 {
					first, last = full[:idx], full[idx+1:]
				} else {
					first = full
				}
			}
		}
		vipVal := strings.ToLower(get(row, cols.vip))
		vip := vipVal == "1" || vipVal == "true" || vipVal == "evet" || vipVal == "yes" || vipVal == "x"

		rows = append(rows, Row{
			Email: email, FirstName: first, LastName: last,
			Position: get(row, cols.position), Department: get(row, cols.department),
			Timezone: get(row, cols.timezone), VIP: vip,
		})
	}
	return rows, errs, nil
}
