package httpapi

import (
	"testing"

	"ai-receptionist/internal/whatsapp"
)

func TestBuildPairingPayloadConnected(t *testing.T) {
	// nil client → not reachable
	snap := buildPairingPayload(nil)
	if snap.Reachable {
		t.Fatal("expected unreachable")
	}

	// Without live WM, only test JSON parity helper
	a := whatsapp.PairingSnapshot{
		Supported:   true,
		Reachable:   true,
		QRAvailable: true,
	}
	b := a
	if !snapshotsEqual(a, b) {
		t.Fatal("identical snapshots should equal")
	}
	a.QRAvailable = false
	if snapshotsEqual(a, b) {
		t.Fatal("different snapshots should not equal")
	}
}

func TestRenderQRPNG(t *testing.T) {
	png, err := renderQRPNGForTest("2@testQRpayload")
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 100 {
		t.Fatalf("png too small: %d bytes", len(png))
	}
}
