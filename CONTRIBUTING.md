# Contributing to AtomGit CLI

首先，感谢你考虑为 AtomGit CLI 做出贡献！

## 如何贡献

### 报告问题

如果你发现了 bug 或有功能建议，请通过以下方式提交：

1. **Bug 报告**：请提供以下信息
   - 使用的操作系统和版本
   - Go 版本 (`go version`)
   - 复现步骤
   - 期望行为 vs 实际行为
   - 错误日志（如果有）

2. **功能建议**：请描述
   - 功能的用例
   - 期望的行为
   - 可能的实现方案（可选）

### 提交代码

1. **Fork 仓库**
   ```bash
   git clone https://atomgit.com/hust-open-atom-club/atomgit-cli.git
   cd atomgit-cli
   ```

2. **创建分支**
   ```bash
   git checkout -b feature/your-feature-name
   # 或
   git checkout -b fix/bug-description
   ```

3. **开发规范**
   - 遵循 Go 代码规范 (gofmt)
   - 添加必要的注释
   - 保持与现有代码风格一致
   - 更新相关文档

4. **测试**
   ```bash
   # 构建项目
   go build ./cmd/ag

   # 运行测试（如果有）
   go test ./...

   # 检查代码格式
   gofmt -l .
   ```

5. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   git push origin feature/your-feature-name
   ```

   提交信息格式：
   - `feat:` 新功能
   - `fix:` 修复 bug
   - `docs:` 文档更新
   - `style:` 代码格式（不影响功能）
   - `refactor:` 代码重构
   - `test:` 测试相关
   - `chore:` 构建过程或辅助工具的变动

6. **创建 Pull Request**
   - 描述更改的内容和原因
   - 关联相关的 issue（如果有）
   - 确保 CI 检查通过

## 开发指南

### 项目结构

```
atomgit-cli/
├── cmd/ag/           # 主程序入口
├── internal/         # 内部包
│   ├── agcmd/        # 核心命令处理
│   ├── config/       # 配置管理
│   └── api/          # API 客户端
├── pkg/              # 公共包
│   ├── cmdutil/      # 命令工具
│   └── cmd/          # 命令实现
└── docs/             # 文档
```

### 添加新命令

参考现有命令的实现模式：

1. 在 `pkg/cmd/<command>/` 目录下创建命令文件
2. 在 `pkg/cmd/root/root.go` 中注册命令
3. 更新 README.md 添加使用说明
4. 更新 CHANGELOG.md

### API 客户端

- 使用 `internal/api/client.go` 中的 HTTP 客户端
- 在 `internal/api/types.go` 中定义数据类型
- 遵循 RESTful API 设计原则

### 代码风格

- 使用 `gofmt` 格式化代码
- 遵循 [Effective Go](https://golang.org/doc/effective_go.html)
- 函数和类型添加注释
- 错误处理要完整

## 行为准则

- 尊重所有参与者
- 接受建设性的批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

## 许可证

通过贡献代码，你同意你的贡献将在 [木兰宽松许可证第2版](LICENSE) 下发布。

## 联系方式

如有问题，可以通过以下方式联系：

- 提交 Issue
- 发送邮件至项目维护者

再次感谢你的贡献！
