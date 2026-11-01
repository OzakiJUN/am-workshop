# 作品包格式（format 1）

作品包是一个 **zip**，所有文件都放在最外层（不要文件夹），一定有一个 `manifest.json`。AM 客户端安装时会按同样的规则再检查一遍。

## manifest.json

```json
{
  "format": 1,
  "type": "theme",
  "id": "weiqi.sakura",
  "name": "樱花",
  "version": "1.0.0",
  "author": "OzakiJUN",
  "description": "粉色的樱花主题，深色模式。"
}
```

| 字段 | 说明 |
| --- | --- |
| `format` | 写 `1` |
| `type` | `theme`、`portrait`、`persona`、`stickers` 之一 |
| `id` | `作者.作品名`，只用小写字母、数字和 `-`，每段最多 31 个字符。要和 `items/` 里的一样，发布后不能改 |
| `name` | 显示的名字，最多 30 个字 |
| `version` | `主.次.补丁`，比如 `1.2.0`。每次发新版都要加，和 `items/` 里的一样 |
| `author` | 你的 GitHub 用户名（仓库的主人） |
| `description` | 介绍，最多 500 个字（可以不写） |
| `permissions` | 留空。以后程序类作品才用 |

整个包最大 20 MB（`workshop.json` 的 `max_package_mb`），单个文件最大 5 MB。图片可以是 `.png`、`.jpg`、`.jpeg`、`.gif`、`.webp`。文件名不要以 `.` 开头。

## 主题 `theme`

在 manifest.json 里加一段 `theme`，写想改的项（不写的保持用户原来的）：

```json
"theme": {
  "mode": "dark",
  "accent": "#ee5b9a",
  "background": "background.jpg",
  "bgBlur": 6,
  "bgDim": 0.3,
  "fontScale": 1,
  "glassStyle": "frosted",
  "glassOpacity": 0.74,
  "glassBlur": 22
}
```

| 项 | 范围 |
| --- | --- |
| `mode` | `light`、`dark`、`system` |
| `accent` | 颜色，`#rrggbb` |
| `background` | 包里一张图片的文件名，最大 3 MB |
| `bgBlur` | 背景模糊 0–30 |
| `bgDim` | 背景变暗 0–0.8 |
| `fontScale` | 字号 0.85–1.25 |
| `glassStyle` | 面板：`frosted` 磨砂、`clear` 清透、`tinted` 染色、`solid` 不透明 |
| `glassOpacity` | 面板不透明度 0.2–1 |
| `glassBlur` | 面板后面的模糊 0–40 |

包里除了背景图不要放别的文件。

## AI 形象包 `portrait`

一组按情绪命名的图片（文件名 = 情绪名 + 扩展名），一定要有 `neutral`。缺的情绪 AM 会按规则用相近的代替。

| 文件名 | 情绪 | 文件名 | 情绪 |
| --- | --- | --- | --- |
| `neutral` | 平静（必须有） | `scared` | 害怕 |
| `happy` | 开心 | `angry` | 生气 |
| `laugh` | 大笑 | `surprised` | 惊讶 |
| `smug` | 得意 | `thinking` | 思考 |
| `shy` | 害羞 | `confused` | 疑惑 |
| `love` | 喜欢 | `speechless` | 无语 |
| `sad` | 难过 | `sleepy` | 困 |
| `cry` | 哭 | `typing` | 正在打字 |
| `worried` | 担心 | `offline` | 离线 |

建议用透明背景的 PNG 或无损 WebP，所有图大小、位置一致（切换时不会跳）。

## 人格 `persona`

只放一个 `persona.json`：

```json
{
  "intro": "一只爱撒娇的猫娘",
  "system": "你是……（它怎么说话、怎么做事，最多 4000 字）",
  "examples": "我：今天好累\n它：喵～辛苦啦，过来让我蹭蹭",
  "greeting": "喵～你来啦"
}
```

`system` 必须写；`intro` 最多 100 字，`examples` 最多 2000 字（一行一句，「我：」/「它：」开头），`greeting` 最多 200 字。

## 表情包 `stickers`

1–200 张图片（每张最大 2 MB），可以再放一个 `tags.json` 给每张图写标签（每张最多 10 个，每个最多 20 个字）：

```json
{
  "狗头.png": ["狗头", "doge"],
  "开心.gif": ["开心", "哈哈"]
}
```

用户安装时会导入到当前服务器的「我的表情」（默认公开，会进那个服务器的公共表情库）。
