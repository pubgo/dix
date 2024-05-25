# dix
> dix 是一个依赖注入框架

> dix 参考了 [user/dig](https://github.com/uber-go/dig) 的设计, 它能够完成更加复杂的依赖注入管理和namespace依赖隔离


## 功能描述
1. dix 支持依赖循环检测
2. dix 支持 func, struct, map, list 作为注入参数
3. dix 支持 map key 作为 namespace 来进行依赖注入的数据隔离
4. dix 支持 struct 对外提供多组依赖对象
5. dix 支持 struct 依赖嵌套
6. dix Inject 支持 func 和 struct 等多种模式进行数据注入
7. dix 对象提供和注入对于原对象无任何侵入
8. dix 被 [pubgo/lava](https://github.com/pubgo/lava/blob/master/cmds/app/cmd.go) 开发框架依赖
9. dix 具体业务使用 [lava/example](https://github.com/pubgo/lava/blob/master/internal/example/grpc/internal/bootstrap/boot.go)
10. 详情请看 [test example](./example/struct-in/main.go)
