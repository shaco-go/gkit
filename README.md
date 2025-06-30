# GKit - Go语言开发助手工具包

GKit 是一个高效的 Go 语言开发助手工具包，提供以下功能：
- 快速克隆 Go 项目模板
- 智能包管理，无需记住完整的包名即可安装依赖
- 实用工具集合，提升 Go 开发效率

## 安装

```bash
go install github.com/shaco-go/gkit@latest
```

## 使用方法

### 创建新项目

从模板仓库创建一个新的项目：

```bash
gkit new github.com/username/template-repo your-project-name
```

参数说明：
- `github.com/username/template-repo`：模板仓库的地址
- `your-project-name`：要创建的项目名称

可选参数：
- `-v, --verbose`：显示详细输出信息

### 安装依赖包

无需记住完整包名，直接使用关键字搜索并安装：

```bash
gkit get package-name
```

例如：
```bash
gkit get gin
```

系统将搜索并列出包含关键字"gin"的所有包，您可以从列表中选择要安装的包。

### 创建Git标签

自动生成和推送Git标签版本：

```bash
gkit tag [message]
```

参数说明：
- `message`：可选，标签附带的提交信息

可选参数：
- `-v, --version`：版本更新类型 (major|minor|patch)，默认为patch

示例：
```bash
# 创建一个patch版本更新的标签（例如v1.0.0 -> v1.0.1）
gkit tag "Bug修复"

# 创建一个minor版本更新的标签（例如v1.0.0 -> v1.1.0）
gkit tag -v minor "新功能发布"

# 创建一个major版本更新的标签（例如v1.0.0 -> v2.0.0）
gkit tag -v major "重大版本更新"
```

如果仓库中没有任何标签，将自动创建初始版本`v0.0.1`。

## 功能特点

1. **项目模板快速克隆**
   - 自动替换模块名
   - 自动处理依赖关系
   - 删除模板仓库的Git历史

2. **智能包管理**
   - 模糊搜索包名
   - 交互式选择安装
   - 自动处理依赖关系

## 许可证

MIT 