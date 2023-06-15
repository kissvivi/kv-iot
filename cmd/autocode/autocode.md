这是一个 Code Generation 工具，用于根据模板生成代码文件。以下是使用说明：

## Code Generation 使用说明

Code Generation 是一个命令行工具，用于根据模板生成代码文件。

### 用法

```shell
codegen [flags]
```

### 命令行选项

- `--service-template` (必需)：指定 Service 模板文件的路径。
- `--api-template` (必需)：指定 API 模板文件的路径。
- `--output-dir` (可选)：指定生成的代码文件的输出目录，默认为当前目录。
- `--package-name` (可选)：指定生成的代码文件的包名。

### 示例

生成代码文件示例:

```shell
codegen --service-template service_template.txt --api-template api_template.txt --output-dir ./generated --package-name main

.\codeGen.exe codegen  -a api_template.txt -s service_template.txt
```

## 安装
1. 构建可执行文件：`go build -o codegen`
2. 运行：`./codegen`

### 使用说明

1. 准备好 Service 模板文件和 API 模板文件。
2. 运行 `codegen` 命令，并按照提示输入必要的字段值。
3. 生成的代码文件将会存储在指定的输出目录中。

## 代码模板

- Service 模板：[service_template.txt](service_template.txt)
- API 模板：[api_template.txt](api_template.txt)

请确保在使用之前将模板文件放置在适当的位置。

希望这个说明对您有所帮助！如果还有其他问题，请随时提问。