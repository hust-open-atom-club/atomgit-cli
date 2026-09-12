# API 契约测试

契约测试用于提前发现请求路径、成功状态、空响应、分页和 JSON 字段变化。
它补充现有单元测试；不将合成样例视为真实服务已验证的证据，也不声称覆盖全部 OpenAPI。

## 离线检查

```sh
make test-contract
```

默认 `go test ./...` 和现有 CI 会运行相同 fixture 回放，不加载真实凭据、不访问网络。
每个 fixture 均通过生产 v5/v8 客户端执行；封闭的 RoundTripper 核对方法、API 版本、
路径和分页参数，并要求消耗完指定页面。未匹配请求直接失败，不回退到真实网络。
解码后的 DTO 字段也有断言，避免 JSON 标签改动后静默得到零值。

首批 fixture 全部为合成数据，建模依据是已有客户端及回归测试：

| Fixture | 契约 |
| --- | --- |
| `repository` | v5 仓库详情、标识及默认分支字段 |
| `issue-list` | v5 数组分页、多个页面及空尾页、分页响应头 |
| `issue-create` | v5 创建 Issue 的 200/201，以及字符串/数字编号 |
| `related-branches-update` | v5 更新关联分支允许的零字节或 JSON 响应 |
| `workflows` | v8 `total_count` / `workflows` 分页信封及工作流 DTO |
| `artifact-delete` | v8 删除仅接受 204 空响应，离线回放 |
| `workflow-dispatch` | v8 调度接受的 200/201/202/204 及可选响应，离线回放 |

## Fixture 格式

文件位于 `internal/api/testdata/contracts/*.json`，由 `internal/apicontract` 严格加载，
拒绝未知配置字段、重复名称、缺失形状和不合法状态码。主要字段：

- `name`、`source`：稳定名称及来源；合成数据明确写 `synthetic`。脱敏采集注明日期、
  对应 API 文档或复现依据、专用账号权限范围，不写账号身份、token 或原始响应。
- `version`、`method`、`path`：`v5` / `v8`、HTTP 方法和不带版本前缀的仓库路径，
  使用 `{owner}` / `{repo}` 占位符。
- `statuses`：端点允许的成功状态集合，不能用所有 2xx 代替确认过的约定。
- `body_policy`：`required` 必须有 JSON；`optional` 允许零字节或匹配形状的 JSON；
  `empty` 只允许零字节或空白。JSON `null` 不等于零字节响应。
- `shape`：递归描述 `types`、`required`、`properties` 和数组的 `items`。
  类型可为 object、array、string、integer、number、boolean、null；多个类型显式列出。
  只将客户端依赖且确认必有的字段列为 required。未列出的新增字段允许存在，
  可选字段在出现时仍检查类型。数组的每个元素均检查，空数组有效。
- `header_types`：可选分页/限流头的 string 或非负 integer 约定；缺失头不自动判为漂移。
- `examples`：包含名称、分页 query、status、白名单 headers、body。
  省略 body 表示零字节；字符串值表示 JSON 字符串。分页参数只允许正整数 page/per_page。

新增 fixture 必须同时在 `TestAPIContracts` 中添加生产客户端回放及 DTO 断言。
构造针对性反例，证明错误状态、空响应变化、信封变化和字段类型变化不会被静默接受。
不要仅更新 fixture 的期望值来消除测试失败。

## 可选在线 smoke

只在专用测试账号/仓库上显式执行。通过 secret 注入 `ATOMGIT_CONTRACT_TOKEN`，
不要把 token 写入命令参数、仓库文件或 shell 跟踪日志。

```sh
export ATOMGIT_CONTRACT_REPO=your-test-owner/your-test-repo
# ATOMGIT_CONTRACT_TOKEN 由安全的凭据注入方式提供
make test-contract-live
```

此目标启用 `contractlive` build tag 和 `ATOMGIT_CONTRACT_LIVE=1`，并禁用测试缓存。
只选取仓库详情、Issue 第一页、工作流第一页三个 GET 端点；**不会创建、编辑、删除资源或调度工作流**。
测试仓库需允许该专用账号读取仓库、Issue 和 Actions 工作流；没有工作流也可验证空列表信封。
如需验证工作流条目字段，应使用已有工作流的专用仓库。没有条目时无法验证条目内部字段。

请求固定访问 `https://api.atomgit.com`，不读取 CLI 凭据配置或任意自定义 API 地址。
拒绝跳转，每个请求总截止时间 15 秒，响应最多 1 MiB。不重试、不输出请求 URL、
响应正文、头值或底层网络错误，只报告 fixture 名称、HTTP 状态或不匹配的字段路径。
403/404 可能是权限或测试仓库配置问题，应先排除环境问题再判断契约漂移。

普通测试不编译在线入口，即便环境中存在 token 也保持离线。若直接使用 build tag，
仍须显式设置 `ATOMGIT_CONTRACT_LIVE=1`；否则在线测试跳过。
本地可以用以下命令检查入口能编译且不会访问网络：

```sh
ATOMGIT_CONTRACT_LIVE=0 go test -tags=contractlive ./internal/api -run '^TestLiveAPIContracts$' -count=1 -v
```

## 采集、脱敏与更新

1. 先复现失败并核对现有实现、API 文档及权限。在线 smoke 不保存数据；确需采集时，
   仅从专用测试资源获取只读响应，在仓库外的临时位置处理，不使用生产私有仓库。
2. 保留方法、路径模式、状态码、类型、分页信封和空响应语义。
   只保留所需分页/限流头，删除 Authorization、Cookie、Set-Cookie 等。
3. 将所有实际用户数据、仓库名、ID、时间戳、正文、路径中的资源标识和 URL 换成合成值。
   删除 token、密码、授权码、查询串凭据、URL userinfo 及潜在私有信息。
   自动字符串替换不能替代人工审查。写操作使用合成 fixture，不运行真实写操作来取样。
4. 写明来源和验证日期，在审查中说明实际服务变化及客户端如何兼容。
   不粘贴原始响应；确认暂存区和 Git diff 只包含脱敏样例。
5. 更新最小必要契约及生产回放，运行 `make test-contract` 和 `go test ./...`。
   若确认服务改变了字段、状态或分页方式，同时修正客户端与用户文档，不能只放宽检查。

新增在线端点必须显式加入 GET allowlist 并评估其只读语义。写操作契约始终通过离线回放验证。
