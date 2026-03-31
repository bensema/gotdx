package gotdx

import "testing"

func TestNewKeepsMainAndExAddresses(t *testing.T) {
	client := New(
		WithTCPAddress("1.1.1.1:7709"),
		WithExTCPAddress("2.2.2.2:7727"),
	)

	if client.mode != clientModeMain {
		t.Fatalf("unexpected client mode: %d", client.mode)
	}
	if client.opt.TCPAddress != "1.1.1.1:7709" {
		t.Fatalf("unexpected main tcp address: %q", client.opt.TCPAddress)
	}
	if client.opt.ExTCPAddress != "2.2.2.2:7727" {
		t.Fatalf("unexpected ex tcp address: %q", client.opt.ExTCPAddress)
	}
}

func TestNewExUsesExAddressAsPrimary(t *testing.T) {
	client := NewEx(
		WithTCPAddress("1.1.1.1:7709"),
		WithExTCPAddress("2.2.2.2:7727"),
	)

	if client.mode != clientModeEx {
		t.Fatalf("unexpected client mode: %d", client.mode)
	}
	if client.opt.TCPAddress != "2.2.2.2:7727" {
		t.Fatalf("unexpected primary tcp address: %q", client.opt.TCPAddress)
	}
	if client.opt.ExTCPAddress != "2.2.2.2:7727" {
		t.Fatalf("unexpected ex tcp address: %q", client.opt.ExTCPAddress)
	}
}
