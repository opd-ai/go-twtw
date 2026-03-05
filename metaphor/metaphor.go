// Package metaphor maps every Go construct kind to a physical machine-part
// metaphor in the style of David Macaulay's The Way Things Work. Each mapping
// includes a visual icon character, a CSS colour for SVG rendering, a short
// human-readable name, and a rationale that explains the analogy in terms a
// non-programmer can understand.
package metaphor

import (
	"fmt"

	"github.com/opd-ai/go-twtw/analyzer"
)

// MachinePartKind identifies the physical metaphor used for a Go construct.
type MachinePartKind string

const (
	// PartGear represents a toothed gear: transforms inputs into outputs,
	// meshing with other gears to transfer motion (data).
	// Used for: functions.
	PartGear MachinePartKind = "gear"

	// PartLever represents a mechanical lever: amplifies or redirects force
	// (behaviour) applied to a pivot (receiver type).
	// Used for: methods.
	PartLever MachinePartKind = "lever"

	// PartAdapter represents a universal adapter plug: allows components with
	// different shapes to connect, as long as they expose the right contacts.
	// Used for: interfaces.
	PartAdapter MachinePartKind = "adapter"

	// PartContainer represents a storage box or hopper: holds raw materials
	// (state/fields) and keeps them organised.
	// Used for: structs.
	PartContainer MachinePartKind = "container"

	// PartPipe represents a hollow pipe: carries a stream of material from one
	// machine part to another without transforming it.
	// Used for: channel types and channel-typed variables.
	PartPipe MachinePartKind = "pipe"

	// PartBelt represents a conveyor belt: moves multiple work-pieces
	// simultaneously and in parallel.
	// Used for: goroutine-spawning functions.
	PartBelt MachinePartKind = "belt"

	// PartGauge represents a pressure gauge or dial: displays the current value
	// of a quantity and can be read or adjusted.
	// Used for: variables.
	PartGauge MachinePartKind = "gauge"

	// PartWeight represents a calibrated weight on a balance: immovable once
	// placed, it anchors the machine to a fixed reference value.
	// Used for: constants.
	PartWeight MachinePartKind = "weight"
)

// MachinePart describes how a single Go construct is represented visually.
type MachinePart struct {
	// Kind is the physical metaphor category.
	Kind MachinePartKind
	// Name is a short human-readable label (e.g. "Bronze Gear").
	Name string
	// Description explains the visual appearance in one sentence.
	Description string
	// Rationale explains the analogy between the Go construct and the machine part.
	Rationale string
	// Color is the primary CSS hex colour used when drawing this part.
	Color string
	// Icon is a Unicode symbol used in text reports as a quick visual indicator.
	Icon string
}

// Map returns the MachinePart metaphor for the given ConstructInfo.
// The mapping is deterministic: the same construct always yields the same part.
func Map(c analyzer.ConstructInfo) MachinePart {
	// A function that spawns goroutines becomes a conveyor belt instead of a gear,
	// because its defining characteristic is parallel, asynchronous work.
	if (c.Kind == analyzer.KindFunction || c.Kind == analyzer.KindMethod) && c.SpawnsGoroutines {
		return beltPart(c)
	}

	// A variable or named type whose underlying type is a channel becomes a pipe.
	if c.HasChannels && (c.Kind == analyzer.KindVar || c.Kind == analyzer.KindType) {
		return pipePart(c)
	}

	switch c.Kind {
	case analyzer.KindFunction:
		return gearPart(c)
	case analyzer.KindMethod:
		return leverPart(c)
	case analyzer.KindInterface:
		return adapterPart(c)
	case analyzer.KindStruct:
		return containerPart(c)
	case analyzer.KindType:
		return typePart(c)
	case analyzer.KindVar:
		return gaugePart(c)
	case analyzer.KindConst:
		return weightPart(c)
	default:
		return gaugePart(c)
	}
}

// gearPart returns the gear metaphor for a function.
func gearPart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartGear,
		Name:        "Bronze Gear",
		Icon:        "⚙",
		Color:       "#cd7f32",
		Description: "A toothed bronze gear with " + toothCountHint(c) + " teeth.",
		Rationale: "A function transforms inputs into outputs, just as a gear " +
			"meshes with neighbouring gears to transfer and transform rotational " +
			"motion. The number of teeth hints at the function's arity.",
	}
}

// leverPart returns the lever metaphor for a method.
func leverPart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartLever,
		Name:        "Mechanical Lever",
		Icon:        "⇋",
		Color:       "#a0a0a0",
		Description: "A metal lever pivoting on the " + receiverLabel(c) + " fulcrum.",
		Rationale: "A method operates on its receiver, much like a lever amplifies " +
			"or redirects force around a fixed pivot point. The receiver type is " +
			"the fulcrum; the method name is the arm.",
	}
}

// adapterPart returns the adapter metaphor for an interface.
func adapterPart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartAdapter,
		Name:        "Universal Adapter",
		Icon:        "🔌",
		Color:       "#d4a017",
		Description: "A golden hexagonal adapter socket with " + methodCountLabel(c) + ".",
		Rationale: "An interface is a universal adapter: any component that " +
			"exposes the required contact points (methods) can be plugged in, " +
			"regardless of its internal construction.",
	}
}

// containerPart returns the container metaphor for a struct.
func containerPart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartContainer,
		Name:        "Storage Container",
		Icon:        "📦",
		Color:       "#7a9e7e",
		Description: "A green sheet-metal box with " + fieldCountLabel(c) + ".",
		Rationale: "A struct is a storage container: it bundles related fields " +
			"(raw materials) together in a single labelled compartment, ready to " +
			"be fed into the next stage of the machine.",
	}
}

// pipePart returns the pipe metaphor for a channel-typed construct.
func pipePart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartPipe,
		Name:        "Steel Pipe",
		Icon:        "≋",
		Color:       "#4682b4",
		Description: "A blue cylindrical pipe with flanged end-caps.",
		Rationale: "A channel is a pipe: it carries a stream of values from one " +
			"part of the machine to another without transforming them, and the " +
			"direction of flow matches the channel direction (send/recv/both).",
	}
}

// beltPart returns the conveyor-belt metaphor for a goroutine-spawning function.
func beltPart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartBelt,
		Name:        "Conveyor Belt",
		Icon:        "⟶",
		Color:       "#606060",
		Description: "A dark-grey rubber conveyor belt with two drive rollers.",
		Rationale: "A function that launches goroutines is a conveyor belt: it " +
			"starts multiple work-pieces moving in parallel, each carried forward " +
			"independently by the belt at the same time.",
	}
}

// gaugePart returns the gauge metaphor for a variable.
func gaugePart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartGauge,
		Name:        "Pressure Gauge",
		Icon:        "🔆",
		Color:       "#deb887",
		Description: "A circular brass gauge dial with a movable needle.",
		Rationale: "A variable is a gauge: it shows the current value of some " +
			"quantity in the machine and that value can change as the machine runs.",
	}
}

// weightPart returns the weight metaphor for a constant.
func weightPart(c analyzer.ConstructInfo) MachinePart {
	return MachinePart{
		Kind:        PartWeight,
		Name:        "Calibrated Weight",
		Icon:        "⚖",
		Color:       "#708090",
		Description: "A slate-grey cast-iron weight with the value stamped on its face.",
		Rationale: "A constant is a calibrated weight: once placed on the balance " +
			"it never moves, providing a fixed reference against which other " +
			"quantities in the machine are compared.",
	}
}

// typePart returns the appropriate metaphor for a named non-struct, non-interface type.
func typePart(c analyzer.ConstructInfo) MachinePart {
	if c.HasChannels {
		return pipePart(c)
	}
	return MachinePart{
		Kind:        PartGauge,
		Name:        "Type Label",
		Icon:        "🏷",
		Color:       "#c8a87a",
		Description: "A copper name-plate affixed to a machine component.",
		Rationale: "A named type gives a familiar label to an underlying " +
			"mechanism, the same way a name-plate identifies the purpose of a " +
			"machine part without changing how it works.",
	}
}

// toothCountHint returns a tooth-count hint based on the number of parameters.
func toothCountHint(c analyzer.ConstructInfo) string {
	n := len(c.Params)
	switch {
	case n == 0:
		return "6"
	case n <= 2:
		return "8"
	case n <= 4:
		return "10"
	default:
		return "12"
	}
}

// receiverLabel returns a human-readable label for the method receiver type.
func receiverLabel(c analyzer.ConstructInfo) string {
	if c.Receiver == "" {
		return "unnamed"
	}
	return c.Receiver
}

// methodCountLabel formats the number of interface methods for a description.
func methodCountLabel(c analyzer.ConstructInfo) string {
	n := len(c.Methods)
	switch n {
	case 0:
		return "no contacts"
	case 1:
		return "1 contact pin"
	default:
		return fmt.Sprintf("%d contact pins", n)
	}
}

// fieldCountLabel formats the number of struct fields for a description.
func fieldCountLabel(c analyzer.ConstructInfo) string {
	n := len(c.Fields)
	switch n {
	case 0:
		return "no compartments"
	case 1:
		return "1 compartment"
	default:
		return fmt.Sprintf("%d compartments", n)
	}
}
