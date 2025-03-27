# 贡献指南

感谢你考虑为Phantasm微服务框架做出贡献！本文档提供了如何参与贡献的指导。

## 行为准则

参与本项目的所有贡献者都需要遵守友好、包容和尊重的交流原则。

## 贡献方式

有多种方式可以为Phantasm做出贡献：

1. **报告Bug**：如果你发现了Bug，请在GitHub上提交issue，并详细描述问题和复现步骤。
2. **提供新功能建议**：如果你有改进Phantasm的想法，也可以通过issue提出。
3. **贡献代码**：通过Pull Request提交代码修复或新功能。
4. **改进文档**：帮助我们完善文档，使其更清晰、更全面。
5. **分享体验**：在博客或社交媒体上分享你使用Phantasm的经验。

## 开发流程

以下是参与代码贡献的基本流程：

### 1. Fork仓库

首先，在GitHub上Fork Phantasm仓库到你的账户。

### 2. 克隆仓库

```bash
git clone https://github.com/dormoron/Phantasm.git
cd Phantasm
git remote add upstream https://github.com/dormoron/Phantasm.git
```

### 3. 创建分支

```bash
git checkout -b feature/your-feature-name
```

命名建议:
- `feature/xxx`：新功能
- `fix/xxx`：Bug修复
- `docs/xxx`：文档改进
- `refactor/xxx`：代码重构

### 4. 开发

进行你的开发工作，确保遵循以下原则：

- 遵循Go语言编码规范
- 添加适当的测试
- 保持代码风格一致
- 保持提交信息清晰

### 5. 提交

```bash
git add .
git commit -m "feat: add some feature" # 遵循Conventional Commits规范
```

我们使用[Conventional Commits](https://www.conventionalcommits.org/)规范进行提交信息格式化。

### 6. 同步上游

```bash
git fetch upstream
git rebase upstream/main
```

### 7. 推送

```bash
git push origin feature/your-feature-name
```

### 8. 提交Pull Request

在GitHub上创建一个从你的分支到Phantasm主仓库main分支的Pull Request。

## 代码审查

所有Pull Request在合并前都会经过代码审查。审查过程中可能会提出修改建议，请及时响应。

## 测试

提交代码时，请确保：

1. 添加了单元测试
2. 所有测试都能通过
3. 代码覆盖率不降低

运行测试：

```bash
go test -v ./...
```

## 文档

如果你修改了公开API或添加了新功能，请同时更新相关文档。好的文档对于用户理解和使用框架至关重要。

## 许可证

通过贡献代码，你同意你的贡献将在MIT许可证下发布。

## 问题与帮助

如果你在贡献过程中有任何问题，欢迎在GitHub上提issue或在讨论区寻求帮助。

感谢你对Phantasm微服务框架的贡献！ 