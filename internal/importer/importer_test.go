package importer

import "testing"

func TestParseCSVCombinedFullName(t *testing.T) {
	csv := "Ad Soyad,E-posta,Departman,Pozisyon\nAyşe Yılmaz,ayse@acme.com,Finans,Uzman\nMehmet Kaya,mehmet@acme.com,IT,Yönetici\n"
	rows, errs, err := ParseFile("targets.csv", []byte(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected row errors: %v", errs)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Email != "ayse@acme.com" || rows[0].FirstName != "Ayşe" || rows[0].LastName != "Yılmaz" || rows[0].Department != "Finans" {
		t.Errorf("unexpected row 0: %+v", rows[0])
	}
}

func TestParseCSVSemicolonDelimiter(t *testing.T) {
	csv := "Email;First Name;Last Name\nbob@acme.com;Bob;Smith\n"
	rows, _, err := ParseFile("targets.csv", []byte(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].Email != "bob@acme.com" || rows[0].FirstName != "Bob" || rows[0].LastName != "Smith" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestParseCSVMissingEmailColumn(t *testing.T) {
	csv := "Ad,Soyad\nAli,Veli\n"
	_, _, err := ParseFile("targets.csv", []byte(csv))
	if err == nil {
		t.Fatal("expected an error when no email column is present")
	}
}

func TestParseCSVInvalidEmailRowSkipped(t *testing.T) {
	csv := "Email,Name\nnot-an-email,Foo\nreal@acme.com,Bar\n"
	rows, errs, err := ParseFile("targets.csv", []byte(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].Email != "real@acme.com" {
		t.Fatalf("expected only the valid row, got: %+v", rows)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 parse error, got %d: %v", len(errs), errs)
	}
}

func TestParseCSVVIPFlag(t *testing.T) {
	csv := "Email,VIP\nceo@acme.com,evet\nstaff@acme.com,hayir\n"
	rows, _, err := ParseFile("targets.csv", []byte(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 || !rows[0].VIP || rows[1].VIP {
		t.Fatalf("unexpected VIP parsing: %+v", rows)
	}
}
