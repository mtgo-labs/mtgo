// compiler/tlgen/testdata/minimal.tl

---types---

// A simple type
testObj#12345678 val:int = TestObj;

// Type with flags
testFlags#abcdef00 flags:# val:flags.0?true other:string = TestFlags;

// Type with vector
testVec#11111111 items:Vector<int> = TestVec;

// Type with optional object
testOpt#22222222 flags:# obj:flags.0?TestObj = TestOpt;

// Bare type used inside vectors
testItem#33333333 x:int y:string = TestItem;

---functions---

// A simple function
testFunc#87654321 val:string = TestObj;

// Function with flags
testFuncFlags#00abcdef flags:# opt_val:flags.5?string = TestObj;
