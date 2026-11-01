package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config is workshop.json at the repository root.
type Config struct {
	Maintainers  []string `json:"maintainers"`
	MaxPackageMB int      `json:"max_package_mb"`
}

// Index is what clients download (index.json, signed in index.sig).
type Index struct {
	Format       int     `json:"format"`
	GeneratedAt  int64   `json:"generated_at"`
	MaxPackageMB int     `json:"max_package_mb"`
	Items        []*Item `json:"items"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "check":
		err = cmdCheck(os.Args[2:])
	case "build":
		err = cmdBuild(os.Args[2:])
	case "sign":
		err = cmdSign(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  workshop check [-author LOGIN] [-base REF] [-summary FILE] [changed items/*.json ...]
  workshop build -o index.json
  workshop sign -key-env WORKSHOP_SIGNING_KEY index.json > index.sig
  workshop verify -pub BASE64 index.json index.sig`)
	os.Exit(2)
}

func loadConfig(dir string) (*Config, error) {
	c := &Config{MaxPackageMB: 20}
	b, err := os.ReadFile(filepath.Join(dir, "workshop.json"))
	if err != nil {
		return nil, err
	}
	return c, json.Unmarshal(b, c)
}

func (c *Config) maintainer(login string) bool {
	for _, m := range c.Maintainers {
		if strings.EqualFold(m, login) {
			return true
		}
	}
	return false
}

func readItem(file string) (*Item, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return parseItem(b)
}

func parseItem(b []byte) (*Item, error) {
	var it Item
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&it); err != nil {
		return nil, fmt.Errorf("不是正确的 JSON（或者有不认识的字段）：%v", err)
	}
	it.SHA256 = strings.ToLower(it.SHA256)
	return &it, nil
}

func itemFiles(dir string) ([]string, error) {
	return filepath.Glob(filepath.Join(dir, "items", "*.json"))
}

var httpClient = &http.Client{Timeout: 3 * time.Minute}

// download fetches a package (at most max bytes).
func download(url string, max int64) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败：HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("包太大了（最多 %d MB）", max>>20)
	}
	return b, nil
}

// checkDownload downloads an item's package and checks size, SHA-256 and contents.
func checkDownload(it *Item, maxMB int) []string {
	data, err := download(it.URL, int64(maxMB)<<20)
	if err != nil {
		return []string{"下载 " + it.URL + "：" + err.Error()}
	}
	var errs []string
	if int64(len(data)) != it.Size {
		errs = append(errs, fmt.Sprintf("size 写的是 %d，下载到的是 %d 字节", it.Size, len(data)))
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != it.SHA256 {
		errs = append(errs, "sha256 对不上：下载到的文件是 "+got)
	}
	return append(errs, CheckPackage(it, data)...)
}

// baseItem reads an item file as it is on the base branch (nil if new).
func baseItem(base, file string) *Item {
	if base == "" {
		return nil
	}
	out, err := exec.Command("git", "show", base+":"+filepath.ToSlash(file)).Output()
	if err != nil {
		return nil
	}
	it, err := parseItem(out)
	if err != nil {
		return nil
	}
	return it
}

func cmdCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	author := fs.String("author", "", "GitHub login of who opened the pull request")
	base := fs.String("base", "", "git ref of the base branch (for who owns changed items)")
	summary := fs.String("summary", "", "write a Markdown report here (e.g. $GITHUB_STEP_SUMMARY)")
	fs.Parse(args)
	cfg, err := loadConfig(".")
	if err != nil {
		return err
	}
	report := map[string][]string{}
	var order []string
	add := func(file string, errs []string) {
		if _, ok := report[file]; !ok {
			order = append(order, file)
		}
		report[file] = append(report[file], errs...)
	}
	// Every item file must stay valid.
	all, err := itemFiles(".")
	if err != nil {
		return err
	}
	ids := map[string]string{}
	for _, f := range all {
		it, err := readItem(f)
		if err != nil {
			add(f, []string{err.Error()})
			continue
		}
		if other, dup := ids[strings.ToLower(it.ID)]; dup {
			add(f, []string{"和 " + other + " 的 id 重复了"})
		}
		ids[strings.ToLower(it.ID)] = f
		if errs := CheckItem(it, f); len(errs) > 0 {
			add(f, errs)
		}
	}
	// The changed ones: who may change them, and their packages.
	for _, f := range fs.Args() {
		if !strings.HasPrefix(filepath.ToSlash(f), "items/") || !strings.HasSuffix(f, ".json") {
			// The tools, workflows and docs are the maintainers' to change.
			if *author != "" && !cfg.maintainer(*author) {
				add(f, []string{"只能改 items/ 里自己作品的文件（别的文件只有工坊管理员能改）"})
			}
			continue
		}
		old := baseItem(*base, f)
		it, err := readItem(f)
		if errors.Is(err, os.ErrNotExist) {
			if *author != "" && !cfg.maintainer(*author) && (old == nil || !strings.EqualFold(old.Author, *author)) {
				add(f, []string{"只有作者自己或工坊管理员能删掉这个作品"})
			} else {
				add(f, nil)
			}
			continue
		}
		if err != nil {
			continue // reported above
		}
		var errs []string
		if *author != "" && !cfg.maintainer(*author) {
			if !strings.EqualFold(it.Author, *author) {
				errs = append(errs, fmt.Sprintf("只能提交自己的作品：author 写的是 %s，提交的人是 %s", it.Author, *author))
			}
			if old != nil && !strings.EqualFold(old.Author, it.Author) {
				errs = append(errs, "不能改别人作品的作者")
			}
			if it.Disabled != "" && (old == nil || old.Disabled != it.Disabled) {
				errs = append(errs, "disabled（下架）只有工坊管理员能写")
			}
		}
		if len(CheckItem(it, f)) == 0 {
			errs = append(errs, checkDownload(it, cfg.MaxPackageMB)...)
		}
		add(f, errs)
	}

	var md strings.Builder
	failed := false
	md.WriteString("## 工坊检查\n\n")
	sort.Strings(order)
	for _, f := range order {
		if len(report[f]) == 0 {
			fmt.Fprintf(&md, "- ✅ `%s`\n", f)
			continue
		}
		failed = true
		fmt.Fprintf(&md, "- ❌ `%s`\n", f)
		for _, e := range report[f] {
			fmt.Fprintf(&md, "  - %s\n", e)
		}
	}
	if len(order) == 0 {
		md.WriteString("没有改动作品。\n")
	}
	if failed {
		md.WriteString("\n有问题的地方改好后再推一次，会重新检查。格式见 FORMAT.md，发布步骤见 PUBLISHING.md。\n")
	} else {
		md.WriteString("\n都没问题，等管理员看过后合并就上架了。\n")
	}
	fmt.Print(md.String())
	if *summary != "" {
		f, err := os.OpenFile(*summary, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			f.WriteString(md.String())
			f.Close()
		}
	}
	if failed {
		return errors.New("check failed")
	}
	return nil
}

// BuildIndex reads every item into the index (sorted by id).
func BuildIndex(dir string, now time.Time) (*Index, error) {
	cfg, err := loadConfig(dir)
	if err != nil {
		return nil, err
	}
	files, err := itemFiles(dir)
	if err != nil {
		return nil, err
	}
	idx := &Index{Format: 1, GeneratedAt: now.UnixMilli(), MaxPackageMB: cfg.MaxPackageMB, Items: []*Item{}}
	for _, f := range files {
		it, err := readItem(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if errs := CheckItem(it, f); len(errs) > 0 {
			return nil, fmt.Errorf("%s: %s", f, strings.Join(errs, "; "))
		}
		idx.Items = append(idx.Items, it)
	}
	sort.Slice(idx.Items, func(i, j int) bool { return idx.Items[i].ID < idx.Items[j].ID })
	return idx, nil
}

func cmdBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	out := fs.String("o", "index.json", "output file")
	fs.Parse(args)
	idx, err := BuildIndex(".", time.Now())
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(idx, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(b, '\n'), 0o644)
}

// keyFromSeed reads the private key the way am-server workshop-keygen writes it.
func keyFromSeed(text string) (ed25519.PrivateKey, error) {
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("the signing key must be the 32-byte seed in base64 (from am-server workshop-keygen)")
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func cmdSign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	env := fs.String("key-env", "WORKSHOP_SIGNING_KEY", "environment variable with the private key")
	fs.Parse(args)
	if fs.NArg() != 1 {
		usage()
	}
	text := os.Getenv(*env)
	if text == "" {
		return fmt.Errorf("%s is empty: add the GitHub Secret first (PUBLISHING.md)", *env)
	}
	key, err := keyFromSeed(text)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	fmt.Println(base64.StdEncoding.EncodeToString(ed25519.Sign(key, data)))
	return nil
}

func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pub := fs.String("pub", "", "public key, base64")
	fs.Parse(args)
	if fs.NArg() != 2 {
		usage()
	}
	key, err := base64.StdEncoding.DecodeString(*pub)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("bad public key")
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	sigText, err := os.ReadFile(fs.Arg(1))
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigText)))
	if err != nil || !ed25519.Verify(key, data, sig) {
		return errors.New("signature does not match")
	}
	fmt.Println("signature OK")
	return nil
}
