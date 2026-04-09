package crypto

import "testing"

func TestEncryptDecryptPassword(t *testing.T) {
	msg := `{"apiKey":"abc"}`
	enc, err := Encrypt(msg, "pw123", "machine-a")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	out, err := Decrypt(enc, "pw123", "machine-a")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if out != msg {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestEncryptDecryptMachineMode(t *testing.T) {
	msg := "payload"
	enc, err := Encrypt(msg, "", "machine-a")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	out, err := Decrypt(enc, "", "machine-a")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if out != msg {
		t.Fatalf("roundtrip mismatch")
	}
}
