# AM 创意工坊

[AM](https://github.com/OzakiJUN/AM) 聊天软件的创意工坊索引：大家做的主题、AI 形象包、人格、表情包都登记在这里，AM 客户端里的「工坊」从这里拉列表、下载、安装。

- 作品本身放在**作者自己的仓库**的 Release 里（源码公开），这里只记「是什么、在哪下载、文件的 SHA-256」。
- 上架 = 向这个仓库提一个 PR，加一个 `items/作者.作品名.json`。自动检查通过、管理员看过合并后就上架了。
- 合并后 GitHub Actions 把所有作品汇总成 `index.json` 并签名，放在 [`index` 分支](../../tree/index)。客户端只信签过名的索引，下载后还会核对 SHA-256，被改过的文件装不上。
- 下架：管理员在作品文件里写 `"disabled": "原因"`，客户端同步后会禁用它。

## 现在能发布的作品

| 类型 | `type` | 是什么 |
| --- | --- | --- |
| 主题 | `theme` | 主题色、深浅色、背景图、字号、面板玻璃效果，一键套用 |
| AI 形象包 | `portrait` | 一套按情绪命名的立绘，给自己的 AI 用 |
| 人格 | `persona` | AI 的人设（怎么说话、开场白），新建 AI 时选它 |
| 表情包 | `stickers` | 一组表情和标签，一键导入「我的表情」 |

小游戏、主页组件、插件这类「程序」以后再开放（要先做好沙盒和权限提示）。

## 文档

- [发布说明（作者看这个）](PUBLISHING.md)
- [作品包格式](FORMAT.md)
- [模板](templates/)：每种类型一个，照着改就行

## 仓库里有什么

| 路径 | 是什么 |
| --- | --- |
| `items/` | 每个作品一个文件 |
| `templates/` | 作品包模板 |
| `samples/` | 工坊自己的示例作品（主题「数字迷踪·夜」、人格「推理迷」） |
| `workshop.json` | 管理员名单、单个包的大小上限 |
| `tools/workshop/` | 检查、生成索引、签名的工具（Go） |
| `.github/workflows/check.yml` | PR 的自动检查 |
| `.github/workflows/publish.yml` | 合并后生成并签名索引，推到 `index` 分支 |
| `.github/workflows/samples.yml` | 把 `samples/*.zip` 传到 Release「samples」 |
