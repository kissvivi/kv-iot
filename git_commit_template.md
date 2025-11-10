# Git Commit Message 模板

## 常用Emoji表情符号

### 代码修改
- 🎨 (art): 优化项目结构 / 代码格式
- ⚡️ (zap): 性能提升
- 🔥 (fire): 移除代码或文件
- 🐛 (bug): 修复bug
- 🚑️ (ambulance): 紧急修复
- ♻️ (recycle): 代码重构
- 💩 (poop): 后续要优化的代码

### 功能相关
- ✨ (sparkles): 引入新功能
- 🎉 (tada): 初始化项目
- 🚧 (construction): 建设中 / WIP
- 🚨 (rotating_light): 修复编译器/linter报错
- 🔧 (wrench): 更新配置文件
- 🔨 (hammer): 更新开发脚本

### 文档相关
- 📝 (memo): 更新文档
- 💡 (bulb): 更新代码注释
- 🌐 (globe_with_meridians): 国际化与本地化
- ✏️ (pencil2): 修复错字

### 测试与部署
- ✅ (white_check_mark): 添加或更新测试用例
- 🚀 (rocket): 部署工作
- 🔖 (bookmark): 发布版本 / 创建tag
- 📦️ (package): 更新打包文件

### 依赖管理
- ⬆️ (arrow_up): 依赖版本升级
- ⬇️ (arrow_down): 依赖版本降级
- ➕ (heavy_plus_sign): 添加依赖
- ➖ (heavy_minus_sign): 移除依赖

### 安全相关
- 🔒️ (lock): 修复安全问题
- 🛂 (passport_control): 权限和角色相关工作

### 数据库相关
- 🗃️ (card_file_box): 数据库相关操作

## 使用示例

```
✨ 添加用户认证功能

- 实现JWT令牌生成与验证
- 添加用户登录API
- 集成权限检查中间件
```

## 提交消息格式建议

```
<emoji> <简短描述> (不超过50个字符)

<详细描述> (可选，不超过72个字符每行)

<引用问题> (可选，如 #123)
```

## 设置为默认模板

在项目根目录运行：
```bash
git config commit.template git_commit_template.md
```