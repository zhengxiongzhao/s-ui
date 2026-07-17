## 🚀 已完成功能 (Completed Features)

- [x] **多节点管理架构**：支持 Panel (控制端) / Agent (代理节点) 双重模式运行
- [x] **节点同步机制**：支持分布式节点状态上报与配置下发核心逻辑
- [x] 默认订阅地址和端口修改

## 📝 待办事项 (TODO / Roadmap)

- [ ] 默认 clash 订阅编辑器内容修改
- [ ] 用户管理：可以选择节点（类似入站标签，在订阅中生效）
- [ ] 节点管理：增加自动同步配置项（开关，是否按预设频率同步）

## 🔗 Repository
https://github.com/zhengxiongzhao/s-ui

```sh
# docker build -f Dockerfile.test -t s-ui-test .
ENV GOPROXY=https://goproxy.cn,direct
docker build --network=host -f Dockerfile.test -t sui:test .
docker compose -f docker-compose-test.yml up

http://localhost:2095/app/settings
admin/admin
```


```
.gitattributes
git add --renormalize .
```