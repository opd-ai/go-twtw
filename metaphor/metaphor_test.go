package metaphor_test

import (
	"testing"

	"github.com/opd-ai/go-twtw/analyzer"
	"github.com/opd-ai/go-twtw/metaphor"
)

func makeConstruct(kind analyzer.ConstructKind, name string, opts ...func(*analyzer.ConstructInfo)) analyzer.ConstructInfo {
	c := analyzer.ConstructInfo{
		Name:     name,
		Kind:     kind,
		Exported: true,
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func withGoroutines(c *analyzer.ConstructInfo)   { c.SpawnsGoroutines = true }
func withChannels(c *analyzer.ConstructInfo)     { c.HasChannels = true }
func withReceiver(r string) func(*analyzer.ConstructInfo) {
	return func(c *analyzer.ConstructInfo) { c.Receiver = r }
}
func withParams(n int) func(*analyzer.ConstructInfo) {
	return func(c *analyzer.ConstructInfo) {
		for i := 0; i < n; i++ {
			c.Params = append(c.Params, analyzer.ParamInfo{Type: "int"})
		}
	}
}
func withMethods(names ...string) func(*analyzer.ConstructInfo) {
	return func(c *analyzer.ConstructInfo) { c.Methods = append(c.Methods, names...) }
}
func withFields(names ...string) func(*analyzer.ConstructInfo) {
	return func(c *analyzer.ConstructInfo) {
		for _, n := range names {
			c.Fields = append(c.Fields, analyzer.FieldInfo{Name: n, Type: "int"})
		}
	}
}

func TestMap_Function_Gear(t *testing.T) {
	c := makeConstruct(analyzer.KindFunction, "Process")
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartGear {
		t.Errorf("expected PartGear, got %q", part.Kind)
	}
	if part.Icon == "" {
		t.Error("Icon should not be empty")
	}
	if part.Color == "" {
		t.Error("Color should not be empty")
	}
	if part.Rationale == "" {
		t.Error("Rationale should not be empty")
	}
}

func TestMap_Function_SpawnsGoroutines_Belt(t *testing.T) {
	c := makeConstruct(analyzer.KindFunction, "RunWorker", withGoroutines)
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartBelt {
		t.Errorf("expected PartBelt for goroutine-spawning function, got %q", part.Kind)
	}
}

func TestMap_Method_Lever(t *testing.T) {
	c := makeConstruct(analyzer.KindMethod, "Reset", withReceiver("*Config"))
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartLever {
		t.Errorf("expected PartLever for method, got %q", part.Kind)
	}
	if part.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestMap_Method_SpawnsGoroutines_Belt(t *testing.T) {
	c := makeConstruct(analyzer.KindMethod, "StartAsync", withReceiver("*Worker"), withGoroutines)
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartBelt {
		t.Errorf("expected PartBelt for goroutine-spawning method, got %q", part.Kind)
	}
}

func TestMap_Interface_Adapter(t *testing.T) {
	c := makeConstruct(analyzer.KindInterface, "Doer", withMethods("Do", "Undo"))
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartAdapter {
		t.Errorf("expected PartAdapter for interface, got %q", part.Kind)
	}
}

func TestMap_Struct_Container(t *testing.T) {
	c := makeConstruct(analyzer.KindStruct, "Config", withFields("Host", "Port", "Timeout"))
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartContainer {
		t.Errorf("expected PartContainer for struct, got %q", part.Kind)
	}
}

func TestMap_Var_Gauge(t *testing.T) {
	c := makeConstruct(analyzer.KindVar, "defaultTimeout")
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartGauge {
		t.Errorf("expected PartGauge for var, got %q", part.Kind)
	}
}

func TestMap_Var_Chan_Pipe(t *testing.T) {
	c := makeConstruct(analyzer.KindVar, "resultChan", withChannels)
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartPipe {
		t.Errorf("expected PartPipe for channel var, got %q", part.Kind)
	}
}

func TestMap_Const_Weight(t *testing.T) {
	c := makeConstruct(analyzer.KindConst, "MaxItems")
	part := metaphor.Map(c)
	if part.Kind != metaphor.PartWeight {
		t.Errorf("expected PartWeight for const, got %q", part.Kind)
	}
}

func TestMap_AllFieldsPopulated(t *testing.T) {
	constructs := []analyzer.ConstructInfo{
		makeConstruct(analyzer.KindFunction, "F"),
		makeConstruct(analyzer.KindMethod, "M", withReceiver("T")),
		makeConstruct(analyzer.KindInterface, "I"),
		makeConstruct(analyzer.KindStruct, "S"),
		makeConstruct(analyzer.KindVar, "v"),
		makeConstruct(analyzer.KindConst, "C"),
		makeConstruct(analyzer.KindType, "T"),
		makeConstruct(analyzer.KindFunction, "Go", withGoroutines),
	}
	for _, c := range constructs {
		part := metaphor.Map(c)
		if part.Kind == "" {
			t.Errorf("%s: Kind is empty", c.Name)
		}
		if part.Name == "" {
			t.Errorf("%s: Name is empty", c.Name)
		}
		if part.Rationale == "" {
			t.Errorf("%s: Rationale is empty", c.Name)
		}
		if part.Color == "" {
			t.Errorf("%s: Color is empty", c.Name)
		}
		if part.Icon == "" {
			t.Errorf("%s: Icon is empty", c.Name)
		}
	}
}
