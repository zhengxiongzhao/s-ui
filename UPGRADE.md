
TODO:
- 节点管理：加个功能，自动同步（默认：是， 按现在的频率同步，否，不自动同步）

添加panel、 agent， 多节点管理支持

https://github.com/alireza0/s-ui

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