





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