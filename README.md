### github 下载工具

#### 安装
``go get github.com/welllog/dlhub``

#### 使用
``dlhub -lang Go -query go -dir github -limit 10``

- lang 按语言搜索
- query 搜索关键字
- dir 保存目录
- limit 限制下载的项目数量
- do 行为，[clone|pull]克隆或更新

#### 补充
搜索按 most stars 降序