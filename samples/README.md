# 示例作品

工坊自己的示例，也是照着做的参考：

- `trail-night/` → `trail-night.zip`：主题「数字迷踪·夜」（带背景图）。
- `detective/` → `detective.zip`：人格「推理迷」。

文件夹里是源文件，zip 是打好的包（文件都在 zip 最外层）。合并到 main 后，`samples.yml` 会把 zip 传到 Release「samples」，`items/` 里的条目指向那里。

改了内容要发新版：改 `manifest.json` 的 `version`，重新打 zip，然后改 `items/` 里对应文件的 `version`、`sha256`、`size`（用 `go run ./tools/workshop checkfile items/<id>.json samples/<名字>.zip` 检查，它会告诉你该填的值）。
