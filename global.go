package dix

func (x *dix) Register(param interface{}) { x.register(param) }
func (x *dix) Inject(param interface{})   { x.inject(param) }
func (x *dix) Invoke()                    { x.invoke() }
