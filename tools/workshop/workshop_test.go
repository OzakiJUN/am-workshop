package main

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func pngBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{255, 0, 0, 255})
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func zipOf(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, data := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(data)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func manifest(typ, extra string) []byte {
	return []byte(`{"format":1,"type":"` + typ + `","id":"weiqi.test","name":"测试","version":"1.0.0","author":"OzakiJUN"` + extra + `}`)
}

func item(typ string) *Item {
	return &Item{ID: "weiqi.test", Type: typ, Name: "测试", Author: "OzakiJUN", Repo: "OzakiJUN/am-test", Version: "1.0.0",
		URL: "https://github.com/OzakiJUN/am-test/releases/download/v1.0.0/test.zip", SHA256: strings.Repeat("a", 64), Size: 1, Summary: "测试用"}
}

func TestItem(t *testing.T) {
	if errs := CheckItem(item("theme"), "items/weiqi.test.json"); len(errs) != 0 {
		t.Fatalf("good item: %v", errs)
	}
	bad := item("game")
	bad.ID = "Weiqi Test"
	bad.Author = "someone"
	bad.URL = "https://example.com/x.zip"
	if errs := CheckItem(bad, "items/other.json"); len(errs) < 5 {
		t.Fatalf("bad item passed: %v", errs)
	}
}

func TestPackages(t *testing.T) {
	img := pngBytes()
	good := map[string]map[string][]byte{
		"theme":    {"manifest.json": manifest("theme", `,"theme":{"mode":"dark","accent":"#ee5b9a","background":"bg.png","bgDim":0.3}`), "bg.png": img},
		"portrait": {"manifest.json": manifest("portrait", ""), "neutral.png": img, "happy.webp": img},
		"persona":  {"manifest.json": manifest("persona", ""), "persona.json": []byte(`{"intro":"猫娘","system":"你是一只猫娘","examples":"","greeting":"喵"}`)},
		"stickers": {"manifest.json": manifest("stickers", ""), "a.png": img, "狗头.png": img, "tags.json": []byte(`{"a.png":["开心"]}`)},
	}
	for typ, files := range good {
		if errs := CheckPackage(item(typ), zipOf(t, files)); len(errs) != 0 {
			t.Errorf("%s: %v", typ, errs)
		}
	}
	bad := map[string]map[string][]byte{
		"theme":    {"manifest.json": manifest("theme", `,"theme":{"accent":"red","fontScale":3,"x":1}`), "extra.png": img},
		"portrait": {"manifest.json": manifest("portrait", ""), "happy.png": img, "readme.txt": []byte("x")},
		"persona":  {"manifest.json": manifest("persona", ""), "persona.json": []byte(`{"system":""}`)},
		"stickers": {"manifest.json": manifest("stickers", `,"permissions":["net"]`), "a.txt": []byte("x"), "tags.json": []byte(`{"b.png":["x"]}`)},
	}
	for typ, files := range bad {
		if errs := CheckPackage(item(typ), zipOf(t, files)); len(errs) == 0 {
			t.Errorf("bad %s passed: %v", typ, errs)
		}
	}
	// manifest must match the item; files in folders are refused.
	other := item("theme")
	other.Version = "2.0.0"
	if errs := CheckPackage(other, zipOf(t, good["theme"])); len(errs) == 0 {
		t.Error("version mismatch passed")
	}
	if errs := CheckPackage(item("portrait"), zipOf(t, map[string][]byte{"manifest.json": manifest("portrait", ""), "x/neutral.png": img})); len(errs) == 0 {
		t.Error("folder passed")
	}
	if errs := CheckPackage(item("theme"), []byte("not a zip")); len(errs) != 1 {
		t.Error("not a zip")
	}
}

func TestDownload(t *testing.T) {
	data := zipOf(t, map[string][]byte{"manifest.json": manifest("portrait", ""), "neutral.png": pngBytes()})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()
	it := item("portrait")
	it.URL = srv.URL + "/x.zip"
	sum := sha256.Sum256(data)
	it.SHA256, it.Size = hex.EncodeToString(sum[:]), int64(len(data))
	if errs := checkDownload(it, 20); len(errs) != 0 {
		t.Fatalf("good download: %v", errs)
	}
	it.SHA256 = strings.Repeat("0", 64)
	if errs := checkDownload(it, 20); len(errs) != 1 || !strings.Contains(errs[0], "sha256") {
		t.Fatalf("changed file: %v", errs)
	}
}

func TestBuildAndSign(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "items"), 0o755)
	os.WriteFile(filepath.Join(dir, "workshop.json"), []byte(`{"maintainers":["OzakiJUN"],"max_package_mb":20}`), 0o644)
	os.WriteFile(filepath.Join(dir, "items", "weiqi.test.json"), []byte(`{"id":"weiqi.test","type":"theme","name":"测试","author":"OzakiJUN",
		"repo":"OzakiJUN/am-test","version":"1.0.0","url":"https://github.com/OzakiJUN/am-test/releases/download/v1.0.0/t.zip",
		"sha256":"`+strings.Repeat("A", 64)+`","size":10,"summary":"测试"}`), 0o644)
	idx, err := BuildIndex(dir, time.UnixMilli(1000))
	if err != nil || len(idx.Items) != 1 || idx.Items[0].SHA256 != strings.Repeat("a", 64) || idx.MaxPackageMB != 20 {
		t.Fatalf("build: %v %+v", err, idx)
	}
	seed := bytes.Repeat([]byte{7}, 32)
	key, err := keyFromSeed(base64.StdEncoding.EncodeToString(seed))
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte(`{"format":1}`)
	if !ed25519.Verify(key.Public().(ed25519.PublicKey), msg, ed25519.Sign(key, msg)) {
		t.Fatal("sign / verify")
	}
	if _, err := keyFromSeed("short"); err == nil {
		t.Fatal("bad key accepted")
	}
	// Unknown fields in an item are refused (typos).
	if _, err := parseItem([]byte(`{"id":"a.b","nmae":"x"}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
}
