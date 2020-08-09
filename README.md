### github 下载工具

#### 安装
``go get github.com/welllog/dlhub``

#### 使用
``dlhub -lang Go -query go -dir github -skip 1 -limit 10``

- lang 按语言搜索
- query 搜索关键字
- dir 保存目录
- skip 跳过前面一定数量的项目
- limit 限制下载的项目数量

#### 补充
搜索按 most stars 降序
项目保存路径按语言分录保存