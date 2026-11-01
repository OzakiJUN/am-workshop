# 模板

每个文件夹是一种作品。把文件夹**里面**的文件（不要文件夹本身）打成 zip：

- `theme/`：改 `manifest.json` 里的 `theme`；要背景图就把图片放进来，再在 `theme` 里写 `"background": "文件名"`。
- `portrait/`：放进按情绪命名的图片（`neutral.png` 必须有，其他见 FORMAT.md 的表）。
- `persona/`：改 `persona.json`。
- `stickers/`：放进表情图片，在 `tags.json` 里给每张写标签（文件名要对上），把示例里的 `example.png` 换成你的。

每个模板都要改 `manifest.json` 里的 `id`、`name`、`author`、`description`。打 zip 时选中里面的文件再压缩（Windows：全选 → 右键 → 压缩为 ZIP 文件），不要把外面的文件夹也压进去。
