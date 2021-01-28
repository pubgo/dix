package dix

// dix(func(struct)struct)
// dix(func(struct)map)
// dix(func(struct)ptr)
// dix(func(ptr)ptr)
// dix(func(ptr)struct)
// dix(func(ptr)map)

func (x *dix) handleStructFnStruct() {}
func (x *dix) handleStructFnMap()    {}
func (x *dix) handleStructFnPtr()    {}
func (x *dix) handlePtrFnPtr()       {}
func (x *dix) handlePtrFnStruct()    {}
func (x *dix) handlePtrFnMap()       {}
