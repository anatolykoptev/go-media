package youtube

import (
	"testing"
)

func TestBuildCommandMaxFileSize(t *testing.T) {
	b := &ytdlpBackend{}
	cfg := Config{}

	t.Run("budget set adds max-filesize flag", func(t *testing.T) {
		cmd := b.buildCommand("/tmp/out.mp4", cfg, 50_000_000)
		flags := cmd.GetFlagConfig().ToFlags().FindByID("max_filesize")
		if len(flags) != 1 {
			t.Fatalf("max_filesize flags: got %d, want 1", len(flags))
		}
		raw := flags[0].Raw()
		// Raw() returns ["--max-filesize=SIZE"] or ["--max-filesize", "SIZE"].
		if len(raw) < 2 {
			t.Fatalf("max_filesize raw args: %v, want >=2", raw)
		}
		// The size argument must be the byte budget (suffixless = bytes).
		if raw[1] != "50000000" {
			t.Fatalf("max_filesize size = %q, want 50000000", raw[1])
		}
	})

	t.Run("zero budget omits max-filesize flag", func(t *testing.T) {
		cmd := b.buildCommand("/tmp/out.mp4", cfg, 0)
		flags := cmd.GetFlagConfig().ToFlags().FindByID("max_filesize")
		if len(flags) != 0 {
			t.Fatalf("max_filesize flags: got %d, want 0 (no budget)", len(flags))
		}
	})
}
