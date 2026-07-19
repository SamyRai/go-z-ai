# 站点生成

本项目附带一个自动生成的静态 HTML 站点（位于 `site/`），由一个小型 Go
生成器从 markdown 文档 + GitHub 实时数据渲染而成。

完整的架构说明和如何添加新语言版本，请见
[英文版文档](../en/site-generation.md)。

## 运行

```bash
make site          # 生成到 ./site
make site-offline  # 同上，但不调用 GitHub API
make site-serve    # 生成 + 在 http://localhost:8000 启动本地预览
```

## 技术栈

- **goldmark** 渲染 markdown 为 HTML（已通过 glamour 间接在 `go.sum` 中）。
- 其它部分**仅使用 Go 标准库**。
- **Catppuccin Mocha/Latte 配色**（MIT 许可），已通过 `catppuccin/go` 间接在
  `go.sum` 中。
- 无 JS 框架、无 Web 字体、无外部 CSS 依赖。
