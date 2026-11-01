// Package main is the am-workshop tool: it checks workshop items and their
// packages (pull requests), builds index.json and signs it (main branch).
// The package format is described in FORMAT.md; the AM client checks the
// same rules again when it installs something (client/src/lib/workshopPackage.ts).
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Item is one file in items/: what the index says about a work.
type Item struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Author   string `json:"author"`
	Repo     string `json:"repo"`
	Version  string `json:"version"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Summary  string `json:"summary"`
	Preview  string `json:"preview,omitempty"`
	Disabled string `json:"disabled,omitempty"`
}

// Manifest is manifest.json inside a package.
type Manifest struct {
	Format      int            `json:"format"`
	Type        string         `json:"type"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Author      string         `json:"author"`
	Description string         `json:"description"`
	Theme       map[string]any `json:"theme,omitempty"`
	Permissions []string       `json:"permissions,omitempty"`
}

// Types that can be published now (data only). Program types (game,
// widget, plugin) come later with a sandbox and permission prompts.
var Types = map[string]string{"theme": "主题", "portrait": "AI 形象包", "persona": "人格", "stickers": "表情包"}

// Moods are the portrait keys (AM docs/design.md §9).
var Moods = []string{"neutral", "happy", "laugh", "smug", "shy", "love", "sad", "cry", "worried",
	"scared", "angry", "surprised", "thinking", "confused", "speechless", "sleepy", "typing", "offline"}

const (
	maxImageBytes      = 5 << 20
	maxStickerBytes    = 2 << 20
	maxBackgroundBytes = 3 << 20
	maxStickers        = 200
	maxTagsPerSticker  = 10
	maxTagLen          = 20
)

var (
	reID      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}\.[a-z0-9][a-z0-9-]{0,30}$`)
	reVersion = regexp.MustCompile(`^\d{1,4}\.\d{1,4}\.\d{1,4}$`)
	reRepo    = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}/[A-Za-z0-9._-]{1,100}$`)
	reSHA     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reColor   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

var imageExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

func runes(s string) int { return utf8.RuneCountInString(s) }

// CheckItem checks an items/ file on its own (no download).
func CheckItem(it *Item, file string) []string {
	var errs []string
	bad := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	if !reID.MatchString(it.ID) {
		bad("id「%s」不对：要写成「作者.作品名」，只用小写字母、数字和 -，比如 weiqi.sakura", it.ID)
	}
	if file != "" && path.Base(file) != it.ID+".json" {
		bad("文件名要和 id 一样：%s.json", it.ID)
	}
	if Types[it.Type] == "" {
		bad("type「%s」不对：现在只能是 theme、portrait、persona、stickers", it.Type)
	}
	if n := runes(strings.TrimSpace(it.Name)); n == 0 || n > 30 {
		bad("name 要写，最多 30 个字")
	}
	if n := runes(it.Summary); n == 0 || n > 60 {
		bad("summary（一行介绍）要写，最多 60 个字")
	}
	if !reRepo.MatchString(it.Repo) {
		bad("repo 要写成「GitHub 用户名/仓库名」")
	} else if !strings.EqualFold(strings.Split(it.Repo, "/")[0], it.Author) {
		bad("author 要是仓库的主人（%s）", strings.Split(it.Repo, "/")[0])
	}
	if !reVersion.MatchString(it.Version) {
		bad("version 要写成 1.0.0 这样")
	}
	prefix := "https://github.com/" + it.Repo + "/releases/download/"
	if !strings.HasPrefix(it.URL, prefix) || !strings.HasSuffix(strings.ToLower(it.URL), ".zip") {
		bad("url 要是这个仓库 Release 里的 zip：%s<标签>/<文件名>.zip", prefix)
	}
	if !reSHA.MatchString(it.SHA256) {
		bad("sha256 要是 64 位小写十六进制")
	}
	if it.Size <= 0 {
		bad("size 要写文件大小（字节）")
	}
	if it.Preview != "" && !strings.HasPrefix(it.Preview, "https://raw.githubusercontent.com/"+it.Repo+"/") &&
		!strings.HasPrefix(it.Preview, "https://github.com/"+it.Repo+"/") {
		bad("preview 只能是这个仓库里的图片（raw.githubusercontent.com/%s/...）", it.Repo)
	}
	return errs
}

// CheckPackage checks a downloaded package against its item.
func CheckPackage(it *Item, data []byte) []string {
	var errs []string
	bad := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return []string{"不是 zip 文件：" + err.Error()}
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := f.Name
		if strings.ContainsAny(name, `/\`) || name == ".." || runes(name) > 100 || strings.HasPrefix(name, ".") {
			bad("「%s」：文件都放在 zip 的最外层，不要放文件夹，文件名不要以 . 开头", name)
			continue
		}
		if f.UncompressedSize64 > maxImageBytes+1 {
			bad("「%s」太大了（最多 5 MB）", name)
			continue
		}
		rc, err := f.Open()
		if err != nil {
			bad("「%s」读不出来：%v", name, err)
			continue
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxImageBytes+1))
		rc.Close()
		if err != nil {
			bad("「%s」读不出来：%v", name, err)
			continue
		}
		files[name] = b
	}
	raw, ok := files["manifest.json"]
	if !ok {
		return append(errs, "zip 最外层要有 manifest.json")
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return append(errs, "manifest.json 不是正确的 JSON："+err.Error())
	}
	if m.Format != 1 {
		bad("manifest.json 里 format 要写 1")
	}
	if m.ID != it.ID || m.Type != it.Type || m.Version != it.Version || !strings.EqualFold(m.Author, it.Author) {
		bad("manifest.json 的 id / type / version / author 要和 items 里的一样（%s / %s / %s / %s）", it.ID, it.Type, it.Version, it.Author)
	}
	if n := runes(strings.TrimSpace(m.Name)); n == 0 || n > 30 {
		bad("manifest.json 的 name 要写，最多 30 个字")
	}
	if runes(m.Description) > 500 {
		bad("description 最多 500 个字")
	}
	if len(m.Permissions) > 0 {
		bad("现在的作品不能申请权限（permissions 留空）")
	}
	delete(files, "manifest.json")
	switch m.Type {
	case "theme":
		errs = append(errs, checkTheme(m.Theme, files)...)
	case "portrait":
		errs = append(errs, checkPortrait(files)...)
	case "persona":
		errs = append(errs, checkPersona(files)...)
	case "stickers":
		errs = append(errs, checkStickers(files)...)
	}
	return errs
}

func ext(name string) string { return strings.ToLower(path.Ext(name)) }

func num(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}

func checkTheme(t map[string]any, files map[string][]byte) []string {
	var errs []string
	bad := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	if len(t) == 0 {
		return []string{"主题要在 manifest.json 里写 theme 一段"}
	}
	ranges := map[string][2]float64{"bgBlur": {0, 30}, "bgDim": {0, 0.8}, "fontScale": {0.85, 1.25}, "glassOpacity": {0.2, 1}, "glassBlur": {0, 40}}
	used := map[string]bool{}
	for k, v := range t {
		switch k {
		case "mode":
			if s, _ := v.(string); s != "light" && s != "dark" && s != "system" {
				bad("theme.mode 只能是 light、dark、system")
			}
		case "accent":
			if s, _ := v.(string); !reColor.MatchString(s) {
				bad("theme.accent 要写成 #ee5b9a 这样的颜色")
			}
		case "glassStyle":
			if s, _ := v.(string); s != "frosted" && s != "clear" && s != "tinted" && s != "solid" {
				bad("theme.glassStyle 只能是 frosted、clear、tinted、solid")
			}
		case "background":
			s, _ := v.(string)
			b, ok := files[s]
			if !ok || !imageExt[ext(s)] {
				bad("theme.background 要是 zip 里的一张图片的文件名")
			} else if len(b) > maxBackgroundBytes {
				bad("背景图最多 3 MB")
			}
			used[s] = true
		default:
			r, known := ranges[k]
			if !known {
				bad("theme 里没有「%s」这一项", k)
				continue
			}
			if f, ok := num(v); !ok || f < r[0] || f > r[1] {
				bad("theme.%s 要在 %g 到 %g 之间", k, r[0], r[1])
			}
		}
	}
	for name := range files {
		if !used[name] {
			bad("「%s」没有用到，删掉它", name)
		}
	}
	return errs
}

func checkPortrait(files map[string][]byte) []string {
	var errs []string
	known := map[string]bool{}
	for _, m := range Moods {
		known[m] = true
	}
	seen := map[string]bool{}
	for name := range files {
		key := strings.TrimSuffix(name, path.Ext(name))
		if !imageExt[ext(name)] || !known[key] {
			errs = append(errs, fmt.Sprintf("「%s」：形象包里只能放按情绪命名的图片（%s，.png / .webp / .jpg / .gif）", name, strings.Join(Moods, "、")))
			continue
		}
		if seen[key] {
			errs = append(errs, fmt.Sprintf("「%s」有两张", key))
		}
		seen[key] = true
	}
	if !seen["neutral"] {
		errs = append(errs, "形象包一定要有 neutral（平静）这一张")
	}
	return errs
}

func checkPersona(files map[string][]byte) []string {
	raw, ok := files["persona.json"]
	if !ok || len(files) != 1 {
		return []string{"人格包里只放 persona.json（和 manifest.json）"}
	}
	var p struct {
		Intro    string `json:"intro"`
		System   string `json:"system"`
		Examples string `json:"examples"`
		Greeting string `json:"greeting"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return []string{"persona.json 不是正确的 JSON：" + err.Error()}
	}
	var errs []string
	if strings.TrimSpace(p.System) == "" || runes(p.System) > 4000 {
		errs = append(errs, "persona.json 的 system（它怎么说话、怎么做事）要写，最多 4000 字")
	}
	if runes(p.Intro) > 100 || runes(p.Examples) > 2000 || runes(p.Greeting) > 200 {
		errs = append(errs, "persona.json：intro 最多 100 字，examples 最多 2000 字，greeting 最多 200 字")
	}
	return errs
}

func checkStickers(files map[string][]byte) []string {
	var errs []string
	tags := map[string][]string{}
	if raw, ok := files["tags.json"]; ok {
		if err := json.Unmarshal(raw, &tags); err != nil {
			errs = append(errs, "tags.json 要写成 {\"图片文件名\": [\"标签\", ...]}："+err.Error())
		}
		delete(files, "tags.json")
	}
	n := 0
	for name, b := range files {
		if !imageExt[ext(name)] {
			errs = append(errs, fmt.Sprintf("「%s」不是图片（.png / .jpg / .gif / .webp）", name))
			continue
		}
		if len(b) > maxStickerBytes {
			errs = append(errs, fmt.Sprintf("「%s」太大了（表情最多 2 MB 一张）", name))
		}
		n++
	}
	if n == 0 || n > maxStickers {
		errs = append(errs, fmt.Sprintf("表情包要有 1 到 %d 张图片", maxStickers))
	}
	for name, t := range tags {
		if _, ok := files[name]; !ok {
			errs = append(errs, fmt.Sprintf("tags.json 里的「%s」不在 zip 里", name))
		}
		if len(t) > maxTagsPerSticker {
			errs = append(errs, fmt.Sprintf("「%s」的标签最多 %d 个", name, maxTagsPerSticker))
		}
		for _, x := range t {
			if runes(x) > maxTagLen {
				errs = append(errs, fmt.Sprintf("标签「%s」太长了（最多 %d 个字）", x, maxTagLen))
			}
		}
	}
	return errs
}
