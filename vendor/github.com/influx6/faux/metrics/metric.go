// Package metrics defines a basic structure foundation for handling logs without
// much hassle, allow more different entries to be created.
// Inspired by https://medium.com/@tjholowaychuk/apex-log-e8d9627f4a9a.
package metrics

// Processors implements a single method to process a Entry.
type Processors interface {
	Handle(Entry) error
}

// Metrics defines an interface with a single method for receiving
// new Entry objects.
type Metrics interface {
	Emit(...EntryMod) error
}

// New returns a Metrics object with the provided Augmenters and  Metrics
// implemement objects for receiving metric Entries.
func New(vals ...interface{}) Metrics {
	var mods []EntryMod
	var procs []Processors

	for _, val := range vals {
		switch item := val.(type) {
		case EntryMod:
			mods = append(mods, item)
		case Processors:
			procs = append(procs, item)
		}
	}

	return &metrics{
		processors: procs,
		mod:        Partial(mods...),
	}
}

type metrics struct {
	mod        EntryMod
	processors []Processors
}

// Emit implements the Metrics interface and delivers Entry
// to undeline metrics.
func (m metrics) Emit(mods ...EntryMod) error {
	if len(m.processors) == 0 {
		return nil
	}

	var en Entry
	Apply(&en, mods...)

	if m.mod != nil {
		m.mod(&en)
	}

	// Deliver augmented Entry to child Metrics
	for _, met := range m.processors {
		if err := met.Handle(en); err != nil {
			return err
		}
	}

	return nil
}

// FilterLevel will return a metrics where all Entry will be filtered by their Entry.Level
// if the level giving is greater or equal to the provided, then it will be received by
// the metrics subscribers.
func FilterLevel(l Level, procs ...Processors) Processors {
	return Case(func(en Entry) bool { return en.Level >= l }, procs...)
}

// DoFn defines a function type which takes a giving Entry.
type DoFn func(Entry) error

type fnMetrics struct {
	do DoFn
}

// DoWith returns a Metrics object where all entries are applied to the provided function.
func DoWith(do DoFn) Processors {
	return fnMetrics{
		do: do,
	}
}

// Handle implements the Processors interface and delivers Entry
// to undeline metrics.
func (m fnMetrics) Handle(en Entry) error {
	return m.do(en)
}

// ConditionalProcessors defines a Processor which first validate it's
// ability to process a giving Entry.
type ConditionalProcessors interface {
	Processors
	Can(Entry) bool
}

// FilterFn defines a function type which takes a giving Entry returning a bool to indicate filtering state.
type FilterFn func(Entry) bool

type caseProcessor struct {
	condition FilterFn
	procs     []Processors
}

// Case returns a Processor object with the provided Augmenters and  Metrics
// implemement objects for receiving metric Entries, where entries are filtered
// out based on a provided function.
func Case(fn FilterFn, procs ...Processors) ConditionalProcessors {
	return caseProcessor{
		condition: fn,
		procs:     procs,
	}
}

// Can returns true/false if we can handle giving Entry.
func (m caseProcessor) Can(en Entry) bool {
	return m.condition(en)
}

// Handle implements the Processors interface and delivers Entry
// to undeline metrics.
func (m caseProcessor) Handle(en Entry) error {
	if m.condition(en) {
		for _, proc := range m.procs {
			if err := proc.Handle(en); err != nil {
				return err
			}
		}
	}
	return nil
}

// switchMaster defines that mod out Entry objects based on a provided function.
type switchMaster struct {
	cases []ConditionalProcessors
}

// Switch returns a new instance of a SwitchMaster.
func Switch(conditions ...ConditionalProcessors) Processors {
	return switchMaster{
		cases: conditions,
	}
}

// Handle delivers the giving entry to all available metricss.
func (fm switchMaster) Handle(e Entry) error {
	for _, proc := range fm.cases {
		if proc.Can(e) {
			if err := proc.Handle(e); err != nil {
				return err
			}
		}
	}
	return nil
}
