﻿# HDF5 Go 库

> **HDF5 文件格式的纯 Go 实现** - 无需 CGo

一个现代化的纯 Go 库，用于读取和写入 HDF5 文件，无需 CGo 依赖。兼容 HDF5 2.0.0，生产就绪的读写支持。

---

## ✨ 特性

- ✅ **纯 Go** - 无 CGo，无 C 依赖，跨平台
- ✅ **现代设计** - 使用 Go 1.25+ 最佳实践构建
- ✅ **HDF5 2.0.0 兼容性** - 读写：v0、v2、v3 超级块 | 格式规范 v4.0 带校验和验证
- ✅ **完整数据集读取** - 紧凑、连续、分块布局，支持 GZIP 压缩
- ✅ **丰富的数据类型** - 整数、浮点数、字符串（固定/可变长度）、复合类型
- ✅ **内存高效** - 缓冲区池和智能内存管理
- ✅ **生产就绪** - 读取支持功能完整
- ✍️ **全面的写入支持** - 数据集、组、属性 + 智能重新平衡！

---

## 🚀 快速开始

### 安装

```bash
go get github.com/huangzhengshun/hdf5
```

### 基本用法

```go
package main

import (
    "fmt"
    "log"
    "github.com/huangzhengshun/hdf5"
)

func main() {
    // 打开 HDF5 文件
    file, err := hdf5.Open("data.h5")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // 遍历文件结构
    file.Walk(func(path string, obj hdf5.Object) {
        switch v := obj.(type) {
        case *hdf5.Group:
            fmt.Printf("📁 %s (%d 个子节点)\n", path, len(v.Children()))
        case *hdf5.Dataset:
            fmt.Printf("📊 %s\n", path)
        }
    })
}
```

**输出**:
```
📁 / (2 个子节点)
📊 /temperature
📁 /experiments/ (3 个子节点)
```

[更多示例 →](examples/)

---

## 📚 文档

### 入门指南
- **[安装指南](docs/guides/INSTALLATION.md)** - 安装和验证库
- **[快速开始](docs/guides/QUICKSTART.md)** - 5 分钟入门
- **[读取数据](docs/guides/READING_DATA.md)** - 读取数据集和属性的综合指南

### 参考文档
- **[数据类型指南](docs/guides/DATATYPES.md)** - HDF5 到 Go 类型映射
- **[故障排除](docs/guides/TROUBLESHOOTING.md)** - 常见问题和解决方案
- **[FAQ](docs/guides/FAQ.md)** - 常见问题解答
- **[API 参考](https://pkg.go.dev/github.com/huangzhengshun/hdf5)** - GoDoc 文档

### 高级主题
- **[架构概述](docs/architecture/OVERVIEW.md)** - 内部工作原理
- **[性能调优](docs/guides/PERFORMANCE_TUNING.md)** - B-tree 重新平衡策略，实现最佳性能
- **[重新平衡 API](docs/guides/REBALANCING_API.md)** - 重新平衡选项的完整 API 参考
- **[示例](examples/)** - 工作代码示例（7 个示例，带详细文档）

---

## ⚡ 性能调优

当删除大量属性时，B-tree 可能会变得**稀疏**（浪费磁盘空间，搜索变慢）。本库提供 **4 种重新平衡策略**：

### 1. 默认（无重新平衡）

**删除速度快，但 B-tree 可能变得稀疏**

```go
// 无选项 = 无重新平衡（类似于 HDF5 C 库）
fw, err := hdf5.CreateForWrite("data.h5", hdf5.CreateTruncate)
```

**适用场景**: 仅追加工作负载，小文件（<100MB）

---

### 2. 延迟重新平衡（比立即重新平衡快 10-100 倍）

**批量处理：达到阈值时重新平衡**

```go
fw, err := hdf5.CreateForWrite("data.h5", hdf5.CreateTruncate,
    hdf5.WithLazyRebalancing(
        hdf5.LazyThreshold(0.05),         // 5% 下溢时触发
        hdf5.LazyMaxDelay(5*time.Minute), // 5 分钟后强制重新平衡
    ),
)
```

**适用场景**: 批量删除工作负载，中/大文件（100-500MB）

**性能**: ~2% 开销，偶尔 100-500ms 暂停

---

### 3. 增量重新平衡（零暂停）

**后台处理：在后台 goroutine 中重新平衡**

```go
fw, err := hdf5.CreateForWrite("data.h5", hdf5.CreateTruncate,
    hdf5.WithLazyRebalancing(),  // 前提条件！
    hdf5.WithIncrementalRebalancing(
        hdf5.IncrementalBudget(100*time.Millisecond),
        hdf5.IncrementalInterval(5*time.Second),
    ),
)
defer fw.Close()  // 停止后台 goroutine
```

**适用场景**: 大文件（>500MB），连续操作，TB 级数据

**性能**: ~4% 开销，**零用户可见暂停**

---

### 4. 智能重新平衡（自动驾驶）

**自动调优：库检测工作负载并选择最佳模式**

```go
fw, err := hdf5.CreateForWrite("data.h5", hdf5.CreateTruncate,
    hdf5.WithSmartRebalancing(
        hdf5.SmartAutoDetect(true),
        hdf5.SmartAutoSwitch(true),
    ),
)
```

**适用场景**: 未知工作负载，混合操作，研究环境

**性能**: ~6% 开销，自动适应

---

### 性能对比

| 模式 | 删除速度 | 暂停时间 | 适用场景 |
|------|----------|----------|----------|
| **默认** | 100% (基准) | 无 | 仅追加，小文件 |
| **延迟** | 95%（比立即重新平衡快 10-100 倍！） | 100-500ms 批量 | 批量删除 |
| **增量** | 92% | 无（后台） | 大文件，连续操作 |
| **智能** | 88% | 可变 | 未知工作负载 |

**了解更多**:
- **[性能调优指南](docs/guides/PERFORMANCE_TUNING.md)**: 综合指南，包含基准测试、建议、故障排除
- **[重新平衡 API 参考](docs/guides/REBALANCING_API.md)**: 完整 API 文档
- **[示例](examples/07-rebalancing/)**: 4 个工作示例，演示每种模式

---

## 🎯 当前状态

**HDF5 2.0.0 就绪，库覆盖率 88%+！** 🎉

### ✅ 完全实现
- **文件结构**:
  - 超级块解析（v0、v2、v3），带校验和验证（CRC32）
  - 对象头 v1（传统 HDF5 < 1.8），带续体
  - 对象头 v2（现代 HDF5 >= 1.8），带续体
  - 组（传统符号表 + 现代对象头）
  - B-tree（大文件的叶子节点 + 非叶子节点）
  - 本地堆（字符串存储）
  - 全局堆（可变长度数据）
  - 分形堆（密集属性的直接块） ✨ 新增

- **数据集读取**:
  - 紧凑布局（数据在对象头中）
  - 连续布局（顺序存储）
  - 带 B-tree 索引的分块布局
  - GZIP/Deflate 压缩
  - LZF 压缩（与 h5py/PyTables 兼容） ✨ 新增
  - 压缩数据的过滤器管道

- **数据类型**（读取 + 写入）:
  - **基本类型**: int8-64, uint8-64, float32/64
  - **AI/ML 类型**: FP8（E4M3、E5M2），bfloat16 - 符合 IEEE 754 标准 ✨ 新增
  - **字符串**: 固定长度（null/空格/null 填充），可变长度（通过全局堆）
  - **高级类型**: 数组、枚举、引用（对象/区域）、不透明类型
  - **复合类型**: 类似结构体，支持嵌套成员

- **属性**:
  - 紧凑属性（在对象头中） ✨ 新增
  - 密集属性（分形堆基础） ✨ 新增
  - 组和数据集的属性读取 ✨ 新增
  - 完整属性 API（Group.Attributes(), Dataset.Attributes()） ✨ 新增

- **导航**: 通过 Walk() 进行完整文件树遍历

- **代码质量**:
  - 测试覆盖率：88%+ 库包（目标：>70%） ✅
  - Lint 问题：0（34+ 个 linters） ✅
  - TODO 项：0（全部已解决） ✅
  - 官方 HDF5 测试套件：433 个文件，100% 通过 ✅

- **安全性** ✨ 新增:
  - 4 个 CVE 已修复（CVE-2025-7067、CVE-2025-6269、CVE-2025-2926、CVE-2025-44905） ✅
  - 全面的溢出保护（SafeMultiply、缓冲区验证） ✅
  - 安全限制：1GB 块、64MB 属性、16MB 字符串 ✅
  - 39 个安全测试用例，全部通过 ✅

### ✍️ 写入支持 - 功能完整！
**生产就绪的写入支持，包含所有功能！** ✅

**数据集操作**:
- ✅ 创建数据集（所有布局：连续、分块、紧凑）
- ✅ 写入数据（所有数据类型，包括复合类型）
- ✅ 数据集调整大小，支持无限维度
- ✅ 可变长度数据类型：字符串、不规则数组
- ✅ 压缩（GZIP、Shuffle、Fletcher32）
- ✅ 数组和枚举数据类型
- ✅ 引用和不透明类型
- ✅ 属性写入（密集 & 紧凑存储）
- ✅ 属性修改/删除

**链接**:
- ✅ 硬链接（完全支持）
- ✅ 软链接（符号引用 - 完全支持）
- ✅ 外部链接（跨文件引用 - 完全支持）

**读取增强**:
- ✅ Hyperslab 选择（数据切片）- 快 10-250 倍！
- ✅ 高效的部分数据集读取
- ✅ 步长和块支持
- ✅ 块感知读取（仅读取需要的块）
- ✅ **ChunkIterator API** - 内存高效的大数据集迭代

**验证**:
- ✅ 官方 HDF5 测试套件：100% 通过（378/378 文件）
- ✅ 生产质量已确认

**未来增强**:
- ✅ LZF 过滤器（读取 + 写入，纯 Go） ✨ 新增
- ✅ BZIP2 过滤器（只读，标准库）
- ⚠️ SZIP 过滤器（存根 - 需要 libaec）
- ⚠️ 带互斥锁的线程安全 + SWMR 模式
- ⚠️ 并行 I/O

### ❌ 计划中的功能

**下一步** - 请参阅 [ROADMAP.md](ROADMAP.md) 获取完整的时间线和版本策略。

---

## 🔧 开发

### 要求
- Go 1.25 或更高版本
- 库无外部依赖

### 构建

```bash
# 克隆仓库
git clone https://github.com/huangzhengshun/hdf5.git
cd hdf5

# 运行测试
go test ./...

# 构建示例
go build ./examples/...

# 构建工具
go build ./cmd/...
```

### 测试

```bash
# 运行所有测试
go test ./...

# 使用竞态检测器运行
go test -race ./...

# 运行覆盖率测试
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 🤝 贡献

欢迎贡献！这是一个早期阶段的项目，我们非常欢迎您的帮助。

**贡献前**:
1. 阅读 [CONTRIBUTING.md](CONTRIBUTING.md) - Git 工作流程和开发指南
2. 查看 [open issues](https://github.com/huangzhengshun/hdf5/issues)
3. 查看 [Architecture Overview](docs/architecture/OVERVIEW.md)

**贡献方式**:
- 🐛 报告 bug
- 💡 建议功能
- 📝 改进文档
- 🔧 提交 pull requests
- ⭐ 给项目点赞

---

## 🗺️ 与其他库的比较

| 特性 | 本库 | gonum/hdf5 | go-hdf5/hdf5 |
|------|------|------------|--------------|
| 纯 Go | ✅ 是 | ❌ CGo 包装器 | ✅ 是 |
| 读取 | ✅ 完整 | ✅ 完整 | ❌ 有限 |
| 写入 | ✅ 完整 | ✅ 完整 | ❌ 无 |
| HDF5 1.8+ | ✅ 是 | ⚠️ 有限 | ❌ 无 |
| 高级数据类型 | ✅ 全部 | ✅ 是 | ❌ 无 |
| 测试套件验证 | ✅ 100% (378/378) | ⚠️ 未知 | ❌ 无 |
| 维护状态 | ✅ 活跃 | ⚠️ 缓慢 | ❌ 不活跃 |
| 线程安全 | ⚠️ 用户必须同步* | ⚠️ 有条件 | ❌ 无 |

\* 不同的 `File` 实例是独立的。对同一 `File` 的并发访问需要用户同步（标准 Go 实践）。完整的线程安全（带互斥锁 + SWMR 模式）计划在未来版本中实现。

---

## 📖 HDF5 资源

- [HDF5 格式规范](https://docs.hdfgroup.org/documentation/hdf5/latest/_f_m_t3.html)
- [官方 HDF5 库](https://github.com/HDFGroup/hdf5)
- [HDF Group](https://www.hdfgroup.org/)

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 🙏 致谢

- HDF Group 提供的 HDF5 格式规范
- gonum/hdf5 提供的灵感
- 本项目的所有贡献者

### 特别感谢

**Ancha Baranova 教授** - 如果没有她宝贵的帮助和支持，这个项目不可能完成。她的协助对于将这个库变为现实至关重要。

---

## 📞 支持

- 📖 [文档](docs/) - 架构和指南
- 🐛 [问题追踪器](https://github.com/huangzhengshun/hdf5/issues)
- 💬 [讨论](https://github.com/huangzhengshun/hdf5/discussions) - 社区问答和公告
- 🌐 [HDF Group Forum](https://forum.hdfgroup.org/t/pure-go-hdf5-library-production-release-with-hdf5-2-0-0-compatibility/13584) - 官方 HDF5 社区讨论

---

**状态**: 稳定 - 兼容 HDF5 2.0.0，带安全加固

---

*由 HDF5 Go 社区用 ❤️ 构建*
*获得 [HDF Group Forum](https://forum.hdfgroup.org/t/pure-go-hdf5-library-production-release-with-hdf5-2-0-0-compatibility/13584) 认可* ⭐