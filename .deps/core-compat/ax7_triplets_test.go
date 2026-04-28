package core

import coretest "dappco.re/go"

func TestAX7_NewRegistry_Good(t *coretest.T) {
	reg := NewRegistry[string]()
	coretest.AssertNotNil(t, reg)
	coretest.AssertEqual(t, 0, reg.Len())
}

func TestAX7_NewRegistry_Bad(t *coretest.T) {
	reg := NewRegistry[int]()
	got := reg.Get("missing")
	coretest.AssertFalse(t, got.OK)
	coretest.AssertNil(t, got.Value)
}

func TestAX7_NewRegistry_Ugly(t *coretest.T) {
	reg := NewRegistry[*int]()
	reg.Set("", nil)
	got := reg.Get("")
	coretest.AssertTrue(t, got.OK)
	coretest.AssertNil(t, got.Value)
}

func TestAX7_NewServiceRuntime_Good(t *coretest.T) {
	runtime := NewServiceRuntime(New(), "opts")
	coretest.AssertNotNil(t, runtime)
	coretest.AssertNotNil(t, runtime.Core)
}

func TestAX7_NewServiceRuntime_Bad(t *coretest.T) {
	runtime := NewServiceRuntime[string](nil, "")
	coretest.AssertNotNil(t, runtime)
	coretest.AssertNil(t, runtime.Core)
}

func TestAX7_NewServiceRuntime_Ugly(t *coretest.T) {
	opts := map[string]string{"name": "api"}
	runtime := NewServiceRuntime(New(), opts)
	coretest.AssertEqual(t, "api", runtime.Options["name"])
	coretest.AssertNotNil(t, runtime.Core)
}
