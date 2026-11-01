# 发布说明

## 作者：发布一个作品

1. **做好作品包**：照着 [`templates/`](templates/) 里对应类型的模板改，格式见 [FORMAT.md](FORMAT.md)。在 AM 客户端的「工坊 → 已安装 → 从文件安装」里先装一次试试（会提示「未审核」，正常）。
2. **放到你自己的 GitHub 仓库**：建一个公开仓库（比如 `am-theme-sakura`），把作品的源文件放进去。
3. **发一个 Release**：仓库页面右边 Releases → Create a new release → Tag 填 `v1.0.0` → 把打好的 zip 拖进附件 → Publish。
4. **算 SHA-256 和大小**（装了 Go 的话，可以先写好第 5 步的文件，再用 `go run ./tools/workshop checkfile items/你的id.json 作品.zip` 一次检查完，它会告诉你 sha256 和 size 该填多少）：
   - Windows：在 zip 所在的文件夹打开 PowerShell，运行 `Get-FileHash 作品.zip`（得到的字母改成小写；大小在文件属性里看「大小」括号里的字节数）。
   - macOS / Linux：`shasum -a 256 作品.zip`、`wc -c 作品.zip`。
5. **提 PR**：在这个仓库点 Fork，在你的 fork 里加一个文件 `items/你的id.json`：

   ```json
   {
     "id": "weiqi.sakura",
     "type": "theme",
     "name": "樱花",
     "author": "OzakiJUN",
     "repo": "OzakiJUN/am-theme-sakura",
     "version": "1.0.0",
     "url": "https://github.com/OzakiJUN/am-theme-sakura/releases/download/v1.0.0/sakura.zip",
     "sha256": "（第 4 步算出来的，64 位）",
     "size": 123456,
     "summary": "一行介绍，最多 60 字",
     "preview": "https://raw.githubusercontent.com/OzakiJUN/am-theme-sakura/main/preview.png"
   }
   ```

   `preview`（预览图）可以不写，写的话只能是你这个仓库里的图片。然后向这个仓库发 Pull Request。
6. **等检查**：PR 下面的 `check` 会自动下载你的包，核对 SHA-256 和内容，结果在检查的 Summary 页里。有 ❌ 就照着改，再推一次会重新检查。
7. **等管理员合并**：合并后几分钟内就会出现在大家的工坊里。

**发新版**：发一个新的 Release（比如 `v1.1.0`），再提 PR 改你那个 `items/` 文件的 `version`、`url`、`sha256`、`size`。已经装了的人会看到「有更新」。

只能提交、修改自己的作品（`author` 要是你自己），也只能改 `items/` 里的文件。`disabled`（下架）只有管理员能写。

## 管理员

- **审核**：看 PR 的检查结果是不是全 ✅，再看看作品本身（下载包看一眼、预览图），没问题就 Merge。合并后 `publish` 会自动生成并签名索引。
- **下架**：在那个作品的文件里加 `"disabled": "下架原因"`，提交到 main。客户端同步后会禁用它（已经套用的主题、导入的表情不会被删掉，但不能再安装）。
- **管理员名单、包大小上限**：`workshop.json`。
- **签名密钥**（只做一次）：
  1. 在自己的电脑上，`am-server.exe` 旁边双击 `workshop_keygen.bat`。它会生成 `workshop_signing_key.txt`（**私钥**）和 `workshop_public_key.txt`（公钥）。
  2. 打开 GitHub 上的这个仓库 → Settings → Secrets and variables → Actions → New repository secret。Name 填 `WORKSHOP_SIGNING_KEY`，Secret 里粘贴 `workshop_signing_key.txt` 的全部内容 → Add secret。
  3. 删掉 `workshop_signing_key.txt`（GitHub 里已经存好了；私钥不要发给任何人，也不要发在聊天里）。
  4. 把**公钥**（窗口里显示的那一行，或者 `workshop_public_key.txt` 的内容）发给开发者，写进 AM 客户端。
  5. 到 Actions → publish → Run workflow 跑一次，`index` 分支里就有签过名的索引了。

  没设置密钥之前，`publish` 会跳过并提示，不会出错。
