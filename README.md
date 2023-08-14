# dix
> dix是一个依赖注入框架

> 它参考了dig的设计, 但是它能够完成更加复杂的依赖注入管理和namespace依赖隔离


## 功能描述
1. 依赖注入循环检测
2. dix 支持func, struct, map, list作为注入参数
3. 支持 map key 作为 namespace 来进行依赖注入的数据隔离
4. dix 支持 struct 对外提供多组依赖对象
5. dix 支持 struct 依赖嵌套
6. dix Inject 支持 func 和 struct 等多种模式进行数据注入
7. dix 对象提供和注入对于原对象无任何侵入
8. 详情请看[Example](./example/struct-in/main.go)
